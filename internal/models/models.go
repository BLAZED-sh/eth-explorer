package models

import "time"

const (
	StatusPending  = "pending"
	StatusMined    = "mined"
	StatusDropped  = "dropped"
	StatusReplaced = "replaced"
)

type Tx struct {
	Hash         string  `gorm:"primaryKey;size:66" json:"hash"`
	Sender       string  `gorm:"size:42;index:idx_sender_nonce,priority:1;index" json:"from"`
	Recipient    *string `gorm:"size:42;index" json:"to"` // nil = contract creation
	Nonce        uint64  `gorm:"index:idx_sender_nonce,priority:2" json:"nonce"`
	ValueWei     string  `json:"valueWei"` // exact value; can exceed int64/float64
	ValueEth     float64 `json:"valueEth"`
	GasLimit     uint64  `json:"gasLimit"`
	GasPriceGwei float64 `json:"gasPriceGwei"` // effective price at first-seen base fee
	MaxFeeGwei   float64 `json:"maxFeeGwei"`   // 0 for legacy
	TipGwei      float64 `json:"tipGwei"`
	TxType       uint8   `json:"type"`
	DataSize     int     `json:"dataSize"`
	MethodSig    string  `gorm:"size:10" json:"methodSig"` // "0x" + first 4 input bytes
	Status       string  `gorm:"size:8;index:idx_status_seen,priority:1;default:pending" json:"status"`
	// ContractAddress is the deterministic CREATE address for a contract-creation
	// tx (Recipient == nil), computed from sender+nonce. nil for normal txs.
	// Lets us answer "which tx deployed this contract?" from our own data.
	ContractAddress *string    `gorm:"size:42;index" json:"contractAddress"`
	SeenInMempool   bool       `json:"seenInMempool"` // false for txs first observed inside a block
	FirstSeen       time.Time  `gorm:"index:idx_status_seen,priority:2,sort:desc" json:"firstSeen"`
	BlockNumber     *uint64    `gorm:"index" json:"blockNumber"`
	BlockPosition   *int       `json:"blockPosition"`
	MinedAt         *time.Time `json:"minedAt"`
	ConfirmMs       *int64     `json:"confirmMs"` // mined_at - first_seen, only when SeenInMempool
	ReplacedBy      *string    `gorm:"size:66" json:"replacedBy"`
}

func (Tx) TableName() string { return "txs" }

type Block struct {
	Number      uint64    `gorm:"primaryKey" json:"number"`
	Hash        string    `gorm:"size:66;uniqueIndex" json:"hash"`
	Timestamp   time.Time `json:"timestamp"`
	ReceivedAt  time.Time `json:"receivedAt"`
	TxCount     int       `json:"txCount"`
	KnownCount  int       `json:"knownCount"` // txs we had as pending — mempool visibility
	GasUsed     uint64    `json:"gasUsed"`
	GasLimit    uint64    `json:"gasLimit"`
	BaseFeeGwei float64   `json:"baseFeeGwei"`
	Miner       string    `gorm:"size:42" json:"miner"`
}

type MempoolSnapshot struct {
	ID           uint      `gorm:"primaryKey" json:"-"`
	At           time.Time `gorm:"index" json:"t"`
	PendingCount int       `json:"pending"` // txpool_status pending (node truth)
	QueuedCount  int       `json:"queued"`
	TrackedCount int       `json:"tracked"` // our in-memory pool size
	BaseFeeGwei  float64   `json:"baseFee"`
	TipP10       float64   `json:"tipP10"`
	TipP25       float64   `json:"tipP25"`
	TipP50       float64   `json:"tipP50"`
	TipP75       float64   `json:"tipP75"`
	TipP90       float64   `json:"tipP90"`
	TxPerSec     float64   `json:"txPerSec"`
}
