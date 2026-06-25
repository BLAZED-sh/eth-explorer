package store

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/fipso/blazed-explorer/internal/models"
)

const (
	pruneInterval     = time.Hour
	snapshotRetention = 7 * 24 * time.Hour
)

// RunPruner deletes old rows hourly and checkpoints the WAL so the -wal file
// doesn't grow unbounded under constant writes.
func (s *Store) RunPruner(ctx context.Context, txRetention time.Duration) {
	t := time.NewTicker(pruneInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			done := make(chan struct{})
			s.Enqueue(func(db *gorm.DB) error {
				defer close(done)
				txCut := time.Now().Add(-txRetention)
				res := db.Where("first_seen < ?", txCut).Delete(&models.Tx{})
				if res.Error != nil {
					return res.Error
				}
				txsDeleted := res.RowsAffected

				snapCut := time.Now().Add(-snapshotRetention)
				if err := db.Where("at < ?", snapCut).Delete(&models.MempoolSnapshot{}).Error; err != nil {
					return err
				}
				if err := db.Where("timestamp < ?", txCut).Delete(&models.Block{}).Error; err != nil {
					return err
				}
				if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE);").Error; err != nil {
					return err
				}
				log.Info().Int64("txs_deleted", txsDeleted).Msg("pruned old rows")
				return nil
			})
			select {
			case <-done:
			case <-ctx.Done():
				return
			}
		}
	}
}
