package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/fipso/blazed-explorer/internal/models"
)

const (
	txBatchMax    = 200
	txFlushEvery  = 250 * time.Millisecond
	txChanBuf     = 4096
	cmdChanBuf    = 256
)

type Store struct {
	DB *gorm.DB

	txCh  chan *models.Tx
	cmdCh chan func(db *gorm.DB) error
}

func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	// _pragma DSN params are applied to every pooled connection.
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", path)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&models.Tx{}, &models.Block{}, &models.MempoolSnapshot{}); err != nil {
		return nil, err
	}
	return &Store{
		DB:    db,
		txCh:  make(chan *models.Tx, txChanBuf),
		cmdCh: make(chan func(db *gorm.DB) error, cmdChanBuf),
	}, nil
}

// EnqueueTx queues a pending tx for batched insert. Drops (with a counter in
// the caller's stats) rather than blocking when the writer is saturated.
func (s *Store) EnqueueTx(t *models.Tx) bool {
	select {
	case s.txCh <- t:
		return true
	default:
		return false
	}
}

// Enqueue schedules an arbitrary write on the single writer goroutine. The
// writer flushes any buffered tx inserts first so commands observe them.
func (s *Store) Enqueue(fn func(db *gorm.DB) error) {
	select {
	case s.cmdCh <- fn:
	default:
		log.Warn().Msg("store command queue full; executing inline")
		s.cmdCh <- fn
	}
}

// RunWriter is the single SQLite writer loop. All mutations funnel through
// here; readers query s.DB concurrently (WAL).
func (s *Store) RunWriter(ctx context.Context) {
	tick := time.NewTicker(txFlushEvery)
	defer tick.Stop()

	buf := make([]*models.Tx, 0, txBatchMax)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		err := s.DB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(buf, txBatchMax).Error
		if err != nil {
			log.Error().Err(err).Int("count", len(buf)).Msg("tx batch insert failed")
		}
		buf = buf[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case t := <-s.txCh:
			buf = append(buf, t)
			if len(buf) >= txBatchMax {
				flush()
			}
		case fn := <-s.cmdCh:
			flush()
			if err := fn(s.DB); err != nil {
				log.Error().Err(err).Msg("store command failed")
			}
		case <-tick.C:
			flush()
		}
	}
}
