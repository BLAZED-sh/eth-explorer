package ingest

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/fipso/blazed-explorer/internal/models"
	"github.com/fipso/blazed-explorer/internal/wire"
)

func (f *Feed) runHeads(ctx context.Context) {
	ethWS := ethclient.NewClient(f.wsRPC)
	for ctx.Err() == nil {
		ch := make(chan *types.Header, 16)
		sub, err := ethWS.SubscribeNewHead(ctx, ch)
		if err != nil {
			log.Error().Err(err).Msg("newHeads subscribe failed; retrying")
			sleepCtx(ctx, 2*time.Second)
			continue
		}
		log.Info().Msg("subscribed to newHeads")
		err = f.consumeHeads(ctx, sub, ch)
		log.Warn().Err(err).Msg("heads subscription dropped; resubscribing")
		sleepCtx(ctx, time.Second)
	}
}

func (f *Feed) consumeHeads(ctx context.Context, sub interface {
	Unsubscribe()
	Err() <-chan error
}, ch <-chan *types.Header) error {
	defer sub.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			if err == nil {
				return errors.New("head subscription closed")
			}
			return err
		case head := <-ch:
			if head == nil {
				continue
			}
			f.handleHead(ctx, head)
		}
	}
}

func (f *Feed) handleHead(ctx context.Context, head *types.Header) {
	rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	numArg := hexutil.EncodeBig(head.Number)
	rb, err := f.fetchRawBlock(rctx, numArg)
	if err != nil || rb == nil {
		// Brief retry; some nodes announce a head before the body is queryable.
		sleepCtx(rctx, 500*time.Millisecond)
		rb, err = f.fetchRawBlock(rctx, numArg)
		if err != nil || rb == nil {
			f.stats.blocksErr.Add(1)
			log.Warn().Err(err).Uint64("num", head.Number.Uint64()).Msg("block lookup failed")
			return
		}
	}
	f.applyBlock(rb, time.Now())
	f.stats.blocksOK.Add(1)
}

func (f *Feed) applyBlock(rb *rawBlock, receivedAt time.Time) {
	baseFee := weiToGwei((*big.Int)(rb.BaseFee))
	blockNum := uint64(rb.Number)
	blockTime := time.Unix(int64(rb.Timestamp), 0)

	util := 0.0
	if rb.GasLimit > 0 {
		util = float64(rb.GasUsed) / float64(rb.GasLimit)
	}
	bm := wire.BlockMsg{
		Number:      blockNum,
		Hash:        strings.ToLower(rb.Hash.Hex()),
		Timestamp:   blockTime.Unix(),
		TxCount:     len(rb.Transactions),
		GasUsed:     uint64(rb.GasUsed),
		GasLimit:    uint64(rb.GasLimit),
		Utilization: util,
		BaseFeeGwei: baseFee,
		Miner:       strings.ToLower(rb.Miner.Hex()),
	}

	// Evict mined txs from the in-memory pool and advance the chain view.
	minedHashes := make([]string, 0, len(rb.Transactions))
	f.mu.Lock()
	for i := range rb.Transactions {
		h := rb.Transactions[i].Hash
		minedHashes = append(minedHashes, strings.ToLower(h.Hex()))
		delete(f.pool, h)
	}
	f.baseFee = baseFee
	f.head = blockNum
	f.mu.Unlock()

	// Pre-convert candidate rows for txs we never saw pending, so the writer
	// closure does no RPC work.
	inserts := make(map[string]*models.Tx, len(rb.Transactions))
	positions := make(map[string]int, len(rb.Transactions))
	for i := range rb.Transactions {
		rt := &rb.Transactions[i]
		m := f.rawTxToModel(rt, blockTime)
		m.Status = models.StatusMined
		m.SeenInMempool = false
		m.BlockNumber = &blockNum
		pos := i
		m.BlockPosition = &pos
		m.MinedAt = &receivedAt
		// Effective gas price against this block's base fee, not the stale one.
		m.GasPriceGwei = effGasPrice(rt, baseFee)
		m.TipGwei = tipGwei(rt, baseFee)
		inserts[m.Hash] = m
		positions[m.Hash] = i
	}

	f.st.Enqueue(func(db *gorm.DB) error {
		var knownCount int
		err := db.Transaction(func(tx *gorm.DB) error {
			// Reorg guard: same number, different hash → previous block's txs
			// go back to pending (the sweeper re-resolves them).
			var existing models.Block
			if err := tx.Where("number = ?", blockNum).First(&existing).Error; err == nil {
				if existing.Hash != bm.Hash {
					log.Warn().Uint64("num", blockNum).Str("old", existing.Hash).Str("new", bm.Hash).Msg("reorg detected; resetting block txs")
					if err := tx.Model(&models.Tx{}).Where("block_number = ?", blockNum).
						Updates(map[string]any{
							"status": models.StatusPending, "block_number": nil,
							"block_position": nil, "mined_at": nil, "confirm_ms": nil,
						}).Error; err != nil {
						return err
					}
				}
			}

			// Which block txs do we already have rows for?
			type seenRow struct {
				Hash          string
				FirstSeen     time.Time
				SeenInMempool bool
			}
			existingRows := make(map[string]seenRow, len(inserts))
			hashes := make([]string, 0, len(inserts))
			for h := range inserts {
				hashes = append(hashes, h)
			}
			for _, chunk := range chunkStrings(hashes, 500) {
				var rows []seenRow
				if err := tx.Model(&models.Tx{}).Select("hash, first_seen, seen_in_mempool").
					Where("hash IN ?", chunk).Scan(&rows).Error; err != nil {
					return err
				}
				for _, r := range rows {
					existingRows[r.Hash] = r
				}
			}

			newRows := make([]*models.Tx, 0, len(inserts))
			for h, m := range inserts {
				row, ok := existingRows[h]
				if !ok {
					newRows = append(newRows, m)
					continue
				}
				updates := map[string]any{
					"status":         models.StatusMined,
					"block_number":   blockNum,
					"block_position": positions[h],
					"mined_at":       receivedAt,
				}
				if row.SeenInMempool {
					knownCount++
					updates["confirm_ms"] = receivedAt.Sub(row.FirstSeen).Milliseconds()
				}
				if err := tx.Model(&models.Tx{}).Where("hash = ?", h).Updates(updates).Error; err != nil {
					return err
				}
			}
			if len(newRows) > 0 {
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
					CreateInBatches(newRows, 200).Error; err != nil {
					return err
				}
			}

			block := &models.Block{
				Number:      blockNum,
				Hash:        bm.Hash,
				Timestamp:   blockTime,
				ReceivedAt:  receivedAt,
				TxCount:     bm.TxCount,
				KnownCount:  knownCount,
				GasUsed:     bm.GasUsed,
				GasLimit:    bm.GasLimit,
				BaseFeeGwei: baseFee,
				Miner:       bm.Miner,
			}
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "number"}},
				UpdateAll: true,
			}).Create(block).Error
		})
		if err != nil {
			return err
		}

		// Publish after persisting so KnownCount is DB-accurate and detail
		// pages linked from the event resolve.
		bm.KnownCount = knownCount
		f.mu.Lock()
		f.blocks = append([]wire.BlockMsg{bm}, f.blocks...)
		if len(f.blocks) > blockRingSize {
			f.blocks = f.blocks[:blockRingSize]
		}
		f.mu.Unlock()
		// New base fee landed — drop the cached tiers so the event is fresh.
		f.invalidateGas()
		f.hub.Publish(wire.BlockEvent(bm, minedHashes, f.GasNow()))
		return nil
	})
}

func chunkStrings(in []string, size int) [][]string {
	var out [][]string
	for len(in) > size {
		out = append(out, in[:size])
		in = in[size:]
	}
	if len(in) > 0 {
		out = append(out, in)
	}
	return out
}
