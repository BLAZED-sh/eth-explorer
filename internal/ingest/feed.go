package ingest

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"

	"github.com/fipso/blazed-explorer/internal/models"
	"github.com/fipso/blazed-explorer/internal/store"
	"github.com/fipso/blazed-explorer/internal/wire"
)

const (
	helloTxRing      = 50
	blockRingSize    = 12
	batchFlushEvery  = 500 * time.Millisecond
	batchMaxSize     = 100
	sweepInterval    = 60 * time.Second
	sweepMinAge      = 3 * time.Minute
	sweepBatch       = 200
	snapshotInterval = 10 * time.Second
)

// Broadcaster is the hub interface the feed publishes events through.
type Broadcaster interface {
	Publish(msg []byte)
}

// rawTx is what we unmarshal from full-body pending tx notifications and from
// eth_getBlockByNumber tx lists. Forward-compatible with future tx types
// because we don't go through go-ethereum's strict Transaction parser, and
// `from` comes for free from the node's JSON.
type rawTx struct {
	Hash                 common.Hash     `json:"hash"`
	From                 common.Address  `json:"from"`
	To                   *common.Address `json:"to"`
	Value                *hexutil.Big    `json:"value"`
	Gas                  hexutil.Uint64  `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Nonce                hexutil.Uint64  `json:"nonce"`
	Input                hexutil.Bytes   `json:"input"`
	Type                 hexutil.Uint64  `json:"type"`
	BlockNumber          *hexutil.Big    `json:"blockNumber"`      // null when pending
	TransactionIndex     *hexutil.Uint64 `json:"transactionIndex"` // null when pending
}

// rawBlock is what we unmarshal from eth_getBlockByNumber(num, true).
type rawBlock struct {
	Number       hexutil.Uint64 `json:"number"`
	Hash         common.Hash    `json:"hash"`
	Timestamp    hexutil.Uint64 `json:"timestamp"`
	GasUsed      hexutil.Uint64 `json:"gasUsed"`
	GasLimit     hexutil.Uint64 `json:"gasLimit"`
	BaseFee      *hexutil.Big   `json:"baseFeePerGas"`
	Miner        common.Address `json:"miner"`
	Transactions []rawTx        `json:"transactions"`
}

// poolTx is the in-memory pending pool entry.
type poolTx struct {
	lite      wire.TxLite
	firstSeen time.Time
}

type feedStats struct {
	received  atomic.Uint64
	dropped   atomic.Uint64
	evicted   atomic.Uint64
	blocksOK  atomic.Uint64
	blocksErr atomic.Uint64
}

type Feed struct {
	st      *store.Store
	hub     Broadcaster
	maxPool int

	wsRPC   *rpc.Client // subscriptions only
	httpRPC *rpc.Client // fetches, polls, batch calls
	eth     *ethclient.Client

	chainID uint64

	mu      sync.RWMutex
	baseFee float64 // gwei
	head    uint64
	pool    map[common.Hash]*poolTx
	recent  []wire.TxLite   // ring of last txs for the hello snapshot
	blocks  []wire.BlockMsg // ring of last blocks for the hello snapshot

	pendingCount atomic.Int64 // from txpool_status
	queuedCount  atomic.Int64
	pendingExact atomic.Bool // true when counts come from txpool_status, not the tracked-pool fallback

	// GasNow memoization. computeGasNow copies+sorts the whole pool, so the
	// result is cached briefly to absorb /api/gas request bursts. gasMu must
	// never be taken while holding f.mu (GasNow takes gasMu then f.mu.RLock).
	gasMu    sync.Mutex
	gasCache wire.GasNow
	gasAt    time.Time

	stats feedStats

	pendingCh chan wire.TxLite // feed → hub batcher
}

func New(ctx context.Context, httpURL, wsURL string, maxPool int, st *store.Store, hub Broadcaster) (*Feed, error) {
	httpRPC, err := rpc.DialContext(ctx, httpURL)
	if err != nil {
		return nil, err
	}
	wsRPC, err := rpc.DialContext(ctx, wsURL)
	if err != nil {
		return nil, err
	}
	eth := ethclient.NewClient(httpRPC)

	chainID, err := eth.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	f := &Feed{
		st:        st,
		hub:       hub,
		maxPool:   maxPool,
		wsRPC:     wsRPC,
		httpRPC:   httpRPC,
		eth:       eth,
		chainID:   chainID.Uint64(),
		pool:      make(map[common.Hash]*poolTx, maxPool),
		pendingCh: make(chan wire.TxLite, 4096),
	}
	log.Info().Uint64("chain_id", f.chainID).Msg("eth client connected")

	// Seed head + base fee so the first txs get a real effective gas price.
	if rb, err := f.fetchRawBlock(ctx, "latest"); err == nil && rb != nil {
		f.mu.Lock()
		f.baseFee = weiToGwei((*big.Int)(rb.BaseFee))
		f.head = uint64(rb.Number)
		f.mu.Unlock()
		log.Info().Uint64("block", uint64(rb.Number)).Float64("base_fee_gwei", f.baseFee).Msg("seeded latest block")
	} else if err != nil {
		log.Warn().Err(err).Msg("failed to seed latest block")
	}

	return f, nil
}

func (f *Feed) Run(ctx context.Context) {
	go f.runPending(ctx)
	go f.runHeads(ctx)
	go f.runBatcher(ctx)
	go f.runSweeper(ctx)
	go f.runSnapshots(ctx)
	go f.logStats(ctx)
	<-ctx.Done()
}

func (f *Feed) ChainID() uint64 { return f.chainID }

func (f *Feed) Head() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.head
}

// Hello builds the snapshot frame sent to each newly connected WS client.
func (f *Feed) Hello() []byte {
	f.mu.RLock()
	head := f.head
	blocks := make([]wire.BlockMsg, len(f.blocks))
	copy(blocks, f.blocks)
	txs := make([]wire.TxLite, len(f.recent))
	copy(txs, f.recent)
	f.mu.RUnlock()
	return wire.HelloEvent(f.chainID, head, f.GasNow(), blocks, txs)
}

func (f *Feed) fetchRawBlock(ctx context.Context, numArg string) (*rawBlock, error) {
	var rb *rawBlock
	if err := f.httpRPC.CallContext(ctx, &rb, "eth_getBlockByNumber", numArg, true); err != nil {
		return nil, err
	}
	return rb, nil
}

func (f *Feed) logStats(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			f.mu.RLock()
			poolSize := len(f.pool)
			f.mu.RUnlock()
			log.Info().
				Uint64("received", f.stats.received.Load()).
				Uint64("dropped", f.stats.dropped.Load()).
				Uint64("evicted", f.stats.evicted.Load()).
				Uint64("blocks_ok", f.stats.blocksOK.Load()).
				Uint64("blocks_err", f.stats.blocksErr.Load()).
				Int("pool", poolSize).
				Msg("ingest stats")
		}
	}
}

// ─── conversions ─────────────────────────────────────────────────────────────

func (f *Feed) rawTxToModel(rt *rawTx, now time.Time) *models.Tx {
	f.mu.RLock()
	baseFee := f.baseFee
	f.mu.RUnlock()

	var recipient *string
	var contractAddr *string
	if rt.To != nil {
		s := strings.ToLower(rt.To.Hex())
		recipient = &s
	} else {
		// Contract creation: the CREATE address is deterministic from the
		// sender and nonce, so we can record it without a receipt.
		ca := strings.ToLower(crypto.CreateAddress(rt.From, uint64(rt.Nonce)).Hex())
		contractAddr = &ca
	}
	valueWei := "0"
	if rt.Value != nil {
		valueWei = (*big.Int)(rt.Value).String()
	}
	methodSig := ""
	if len(rt.Input) >= 4 {
		methodSig = hexutil.Encode(rt.Input[:4])
	}
	return &models.Tx{
		Hash:            strings.ToLower(rt.Hash.Hex()),
		Sender:          strings.ToLower(rt.From.Hex()),
		Recipient:       recipient,
		Nonce:           uint64(rt.Nonce),
		ValueWei:        valueWei,
		ValueEth:        weiToEth((*big.Int)(rt.Value)),
		GasLimit:        uint64(rt.Gas),
		GasPriceGwei:    effGasPrice(rt, baseFee),
		MaxFeeGwei:      weiToGwei((*big.Int)(rt.MaxFeePerGas)),
		TipGwei:         tipGwei(rt, baseFee),
		TxType:          uint8(rt.Type),
		DataSize:        len(rt.Input),
		MethodSig:       methodSig,
		Status:          models.StatusPending,
		ContractAddress: contractAddr,
		SeenInMempool:   true,
		FirstSeen:       now,
	}
}

// effGasPrice returns the effective gas price in gwei: min(maxFee, base+tip)
// for EIP-1559 txs, the plain gas price for legacy txs.
func effGasPrice(rt *rawTx, baseFeeGwei float64) float64 {
	if rt.MaxFeePerGas != nil && rt.MaxPriorityFeePerGas != nil {
		tip := weiToGwei((*big.Int)(rt.MaxPriorityFeePerGas))
		feeCap := weiToGwei((*big.Int)(rt.MaxFeePerGas))
		if eff := baseFeeGwei + tip; eff < feeCap {
			return eff
		}
		return feeCap
	}
	return weiToGwei((*big.Int)(rt.GasPrice))
}

// tipGwei returns the priority fee in gwei; for legacy txs it derives an
// effective tip as max(0, gasPrice - baseFee).
func tipGwei(rt *rawTx, baseFeeGwei float64) float64 {
	if rt.MaxPriorityFeePerGas != nil {
		return weiToGwei((*big.Int)(rt.MaxPriorityFeePerGas))
	}
	if rt.GasPrice != nil {
		if t := weiToGwei((*big.Int)(rt.GasPrice)) - baseFeeGwei; t > 0 {
			return t
		}
	}
	return 0
}

func weiToGwei(v *big.Int) float64 {
	if v == nil {
		return 0
	}
	gwei, _ := new(big.Float).Quo(new(big.Float).SetInt(v), big.NewFloat(1e9)).Float64()
	return gwei
}

func weiToEth(v *big.Int) float64 {
	if v == nil {
		return 0
	}
	eth, _ := new(big.Float).Quo(new(big.Float).SetInt(v), big.NewFloat(1e18)).Float64()
	return eth
}
