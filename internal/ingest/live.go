package ingest

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/fipso/blazed-explorer/internal/models"
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
