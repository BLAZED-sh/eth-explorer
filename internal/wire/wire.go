// Package wire defines the JSON shapes shared by the REST API and the
// browser-facing WebSocket events. The frontend mirrors these in types.ts.
package wire

import (
	"encoding/json"

	"github.com/fipso/blazed-explorer/internal/models"
)

type TxLite struct {
	Hash         string  `json:"hash"`
	From         string  `json:"from"`
	To           *string `json:"to"`
	ValueEth     float64 `json:"valueEth"`
	GasLimit     uint64  `json:"gasLimit"`
	GasPriceGwei float64 `json:"gasPriceGwei"`
	TipGwei      float64 `json:"tipGwei"`
	Type         uint8   `json:"type"`
	DataSize     int     `json:"dataSize"`
	MethodSig    string  `json:"methodSig"`
	Status       string  `json:"status"`
	FirstSeen    int64   `json:"firstSeen"` // unix ms
}

func TxLiteFromModel(t *models.Tx) TxLite {
	return TxLite{
		Hash:         t.Hash,
		From:         t.Sender,
		To:           t.Recipient,
		ValueEth:     t.ValueEth,
		GasLimit:     t.GasLimit,
		GasPriceGwei: t.GasPriceGwei,
		TipGwei:      t.TipGwei,
		Type:         t.TxType,
		DataSize:     t.DataSize,
		MethodSig:    t.MethodSig,
		Status:       t.Status,
		FirstSeen:    t.FirstSeen.UnixMilli(),
	}
}

type BlockMsg struct {
	Number      uint64  `json:"number"`
	Hash        string  `json:"hash"`
	Timestamp   int64   `json:"timestamp"` // unix s
	TxCount     int     `json:"txCount"`
	KnownCount  int     `json:"knownCount"`
	GasUsed     uint64  `json:"gasUsed"`
	GasLimit    uint64  `json:"gasLimit"`
	Utilization float64 `json:"utilization"`
	BaseFeeGwei float64 `json:"baseFeeGwei"`
	Miner       string  `json:"miner"`
}

func BlockMsgFromModel(b *models.Block) BlockMsg {
	util := 0.0
	if b.GasLimit > 0 {
		util = float64(b.GasUsed) / float64(b.GasLimit)
	}
	return BlockMsg{
		Number:      b.Number,
		Hash:        b.Hash,
		Timestamp:   b.Timestamp.Unix(),
		TxCount:     b.TxCount,
		KnownCount:  b.KnownCount,
		GasUsed:     b.GasUsed,
		GasLimit:    b.GasLimit,
		Utilization: util,
		BaseFeeGwei: b.BaseFeeGwei,
		Miner:       b.Miner,
	}
}

type GasNow struct {
	BaseFeeGwei  float64 `json:"baseFeeGwei"`
	Slow         float64 `json:"slow"`
	Standard     float64 `json:"standard"`
	Fast         float64 `json:"fast"`
	Rapid        float64 `json:"rapid"`
	PendingCount int     `json:"pendingCount"`
	QueuedCount  int     `json:"queuedCount"`
	PendingExact bool    `json:"pendingExact"` // false when PendingCount is the tracked-pool floor, not node-exact
}

type StatsMsg struct {
	At           int64   `json:"at"` // unix ms
	Pending      int     `json:"pending"`
	Queued       int     `json:"queued"`
	PendingExact bool    `json:"pendingExact"` // false when Pending is the tracked-pool floor, not node-exact
	BaseFeeGwei  float64 `json:"baseFeeGwei"`
	TipP10       float64 `json:"tipP10"`
	TipP50       float64 `json:"tipP50"`
	TipP90       float64 `json:"tipP90"`
	TxPerSec     float64 `json:"txPerSec"`
}

// ─── WS event envelopes ─────────────────────────────────────────────────────

func HelloEvent(chainID uint64, head uint64, gas GasNow, blocks []BlockMsg, txs []TxLite) []byte {
	b, _ := json.Marshal(struct {
		T            string     `json:"t"`
		ChainID      uint64     `json:"chainId"`
		Head         uint64     `json:"head"`
		Gas          GasNow     `json:"gas"`
		RecentBlocks []BlockMsg `json:"recentBlocks"`
		Txs          []TxLite   `json:"txs"`
	}{"hello", chainID, head, gas, blocks, txs})
	return b
}

func PendingBatchEvent(txs []TxLite) []byte {
	b, _ := json.Marshal(struct {
		T   string   `json:"t"`
		Txs []TxLite `json:"txs"`
	}{"pending_batch", txs})
	return b
}

func BlockEvent(block BlockMsg, minedHashes []string, gas GasNow) []byte {
	b, _ := json.Marshal(struct {
		T           string   `json:"t"`
		Block       BlockMsg `json:"block"`
		MinedHashes []string `json:"minedHashes"`
		Gas         GasNow   `json:"gas"`
	}{"block", block, minedHashes, gas})
	return b
}

func StatsEvent(s StatsMsg) []byte {
	b, _ := json.Marshal(struct {
		T string `json:"t"`
		StatsMsg
	}{"stats", s})
	return b
}

// HistoryPoint is one element of /api/stats/history.
type HistoryPoint struct {
	T        int64   `json:"t"` // unix ms
	Pending  int     `json:"pending"`
	Queued   int     `json:"queued"`
	BaseFee  float64 `json:"baseFee"`
	TipP10   float64 `json:"tipP10"`
	TipP50   float64 `json:"tipP50"`
	TipP90   float64 `json:"tipP90"`
	TxPerSec float64 `json:"txPerSec"`
}

func HistoryPointFromSnapshot(s *models.MempoolSnapshot) HistoryPoint {
	return HistoryPoint{
		T:        s.At.UnixMilli(),
		Pending:  s.PendingCount,
		Queued:   s.QueuedCount,
		BaseFee:  s.BaseFeeGwei,
		TipP10:   s.TipP10,
		TipP50:   s.TipP50,
		TipP90:   s.TipP90,
		TxPerSec: s.TxPerSec,
	}
}
