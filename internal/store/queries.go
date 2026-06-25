package store

import (
	"time"

	"github.com/fipso/blazed-explorer/internal/models"
)

// RecentTxs returns txs by status ordered newest-first using keyset pagination
// on first_seen (before = exclusive upper bound, zero value = no bound).
func (s *Store) RecentTxs(status string, limit int, before time.Time) ([]models.Tx, error) {
	q := s.DB.Order("first_seen DESC").Limit(limit)
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	if !before.IsZero() {
		q = q.Where("first_seen < ?", before)
	}
	var txs []models.Tx
	err := q.Find(&txs).Error
	return txs, err
}

func (s *Store) TxByHash(hash string) (*models.Tx, error) {
	var t models.Tx
	if err := s.DB.Where("hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// TxByContractAddress returns the creation tx that deployed the contract at
// addr, if we ingested it (ContractAddress is set only for creation txs).
func (s *Store) TxByContractAddress(addr string) (*models.Tx, error) {
	var t models.Tx
	if err := s.DB.Where("contract_address = ?", addr).
		Order("first_seen DESC").First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) TxsByAddress(addr string, limit int, before time.Time) ([]models.Tx, error) {
	q := s.DB.Where("sender = ? OR recipient = ?", addr, addr).
		Order("first_seen DESC").Limit(limit)
	if !before.IsZero() {
		q = q.Where("first_seen < ?", before)
	}
	var txs []models.Tx
	err := q.Find(&txs).Error
	return txs, err
}

func (s *Store) Blocks(limit int, before uint64) ([]models.Block, error) {
	q := s.DB.Order("number DESC").Limit(limit)
	if before > 0 {
		q = q.Where("number < ?", before)
	}
	var blocks []models.Block
	err := q.Find(&blocks).Error
	return blocks, err
}

func (s *Store) BlockByNumber(num uint64) (*models.Block, error) {
	var b models.Block
	if err := s.DB.Where("number = ?", num).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) BlockTxs(num uint64) ([]models.Tx, error) {
	var txs []models.Tx
	err := s.DB.Where("block_number = ?", num).
		Order("block_position ASC").Find(&txs).Error
	return txs, err
}

// Snapshots returns time-series points within [since, now], oldest-first,
// downsampled in SQL to at most ~maxPoints rows. Snapshots are written at a
// fixed cadence so the auto-increment id is proportional to time; we keep every
// stride-th row (id % stride == 0), which is time-uniform and, crucially, only
// materializes ~maxPoints rows instead of the whole window (a 7d window is
// ~60k rows). maxPoints <= 0 means no downsampling.
func (s *Store) Snapshots(since time.Time, maxPoints int) ([]models.MempoolSnapshot, error) {
	q := s.DB.Where("at >= ?", since)

	if maxPoints > 0 {
		var count int64
		if err := s.DB.Model(&models.MempoolSnapshot{}).Where("at >= ?", since).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > int64(maxPoints) {
			stride := (count + int64(maxPoints) - 1) / int64(maxPoints)
			q = q.Where("id % ? = 0", stride)
		}
	}

	var snaps []models.MempoolSnapshot
	err := q.Order("at ASC").Find(&snaps).Error
	return snaps, err
}

// StalePending returns pending txs older than cutoff, oldest first, for the
// dropped/replaced sweeper.
func (s *Store) StalePending(cutoff time.Time, limit int) ([]models.Tx, error) {
	var txs []models.Tx
	err := s.DB.Where("status = ? AND first_seen < ?", models.StatusPending, cutoff).
		Order("first_seen ASC").Limit(limit).Find(&txs).Error
	return txs, err
}

func (s *Store) TxCount() (int64, error) {
	var n int64
	err := s.DB.Model(&models.Tx{}).Count(&n).Error
	return n, err
}

// MinedTxBySenderNonce finds a mined tx with the same sender+nonce (used to
// distinguish replaced from dropped).
func (s *Store) MinedTxBySenderNonce(sender string, nonce uint64, excludeHash string) (*models.Tx, error) {
	var t models.Tx
	err := s.DB.Where("sender = ? AND nonce = ? AND status = ? AND hash != ?",
		sender, nonce, models.StatusMined, excludeHash).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}
