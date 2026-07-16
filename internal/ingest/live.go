package ingest

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/fipso/blazed-explorer/internal/models"
	"github.com/fipso/blazed-explorer/internal/wire"
)

// FetchTx asks the node for its current view of a tx, returning a model row
// (not persisted) plus the full calldata hex. Returns (nil, "", nil) when the
// node doesn't know the hash.
func (f *Feed) FetchTx(ctx context.Context, hash string) (*models.Tx, string, error) {
	var rt *rawTx
	if err := f.httpRPC.CallContext(ctx, &rt, "eth_getTransactionByHash", hash); err != nil {
		return nil, "", err
	}
	if rt == nil {
		return nil, "", nil
	}
	m := f.rawTxToModel(rt, time.Now())
	m.SeenInMempool = false
	if rt.BlockNumber != nil {
		m.Status = models.StatusMined
		num := rt.BlockNumber.ToInt().Uint64()
		m.BlockNumber = &num
		if rt.TransactionIndex != nil {
			pos := int(*rt.TransactionIndex)
			m.BlockPosition = &pos
		}
	}
	return m, hexutil.Encode(rt.Input), nil
}

// FetchBlock asks the node for a block by number, returning the wire message
// plus tx lites (not persisted). Used for blocks outside the retained DB
// window. Returns (nil, nil, nil) when the node does not know the block.
func (f *Feed) FetchBlock(ctx context.Context, num uint64) (*wire.BlockMsg, []wire.TxLite, error) {
	rb, err := f.fetchRawBlock(ctx, hexutil.EncodeUint64(num))
	if err != nil {
		return nil, nil, err
	}
	if rb == nil {
		return nil, nil, nil
	}

	baseFee := weiToGwei((*big.Int)(rb.BaseFee))
	blockNum := uint64(rb.Number)
	blockTime := time.Unix(int64(rb.Timestamp), 0)

	util := 0.0
	if rb.GasLimit > 0 {
		util = float64(rb.GasUsed) / float64(rb.GasLimit)
	}
	bm := &wire.BlockMsg{
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

	lites := make([]wire.TxLite, len(rb.Transactions))
	for i := range rb.Transactions {
		rt := &rb.Transactions[i]
		m := f.rawTxToModel(rt, blockTime)
		m.Status = models.StatusMined
		m.SeenInMempool = false
		m.BlockNumber = &blockNum
		pos := i
		m.BlockPosition = &pos
		// Effective gas price against this block's base fee, not the head's.
		m.GasPriceGwei = effGasPrice(rt, baseFee)
		m.TipGwei = tipGwei(rt, baseFee)
		lites[i] = wire.TxLiteFromModel(m)
	}
	return bm, lites, nil
}

// FetchCode returns the runtime bytecode deployed at addr as a hex string.
// "0x" means no code (an EOA or a self-destructed / not-yet-deployed contract).
func (f *Feed) FetchCode(ctx context.Context, addr string) (string, error) {
	var code hexutil.Bytes
	if err := f.httpRPC.CallContext(ctx, &code, "eth_getCode", addr, "latest"); err != nil {
		return "", err
	}
	return hexutil.Encode(code), nil
}

// FetchReceipt returns the node's receipt JSON verbatim (null if not mined).
func (f *Feed) FetchReceipt(ctx context.Context, hash string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := f.httpRPC.CallContext(ctx, &raw, "eth_getTransactionReceipt", hash); err != nil {
		return nil, err
	}
	return raw, nil
}
