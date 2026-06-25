package ingest

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/fipso/blazed-explorer/internal/models"
	"github.com/fipso/blazed-explorer/internal/wire"
)

type txpoolStatus struct {
	Pending hexutil.Uint64 `json:"pending"`
	Queued  hexutil.Uint64 `json:"queued"`
}

// runSnapshots polls txpool_status and writes a time-series row every tick,
// publishing the same numbers to browsers as a stats event.
func (f *Feed) runSnapshots(ctx context.Context) {
	t := time.NewTicker(snapshotInterval)
	defer t.Stop()
	var lastReceived uint64
	var lastTick = time.Now()
	var lastTxpoolWarn time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := time.Now()

			f.mu.RLock()
			tracked := len(f.pool)
			f.mu.RUnlock()

			rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			var status txpoolStatus
			if err := f.httpRPC.CallContext(rctx, &status, "txpool_status"); err != nil {
				// Node without the txpool namespace: fall back to the txs we
				// are tracking ourselves so counts/charts don't flatline at 0.
				// This is a floor (capped at maxPool), not the node-exact count.
				f.pendingCount.Store(int64(tracked))
				f.pendingExact.Store(false)
				if time.Since(lastTxpoolWarn) > 10*time.Minute {
					lastTxpoolWarn = time.Now()
					log.Warn().Err(err).Msg("txpool_status unavailable; using tracked pool size as pending count (enable the txpool RPC namespace for node-exact counts)")
				}
			} else {
				f.pendingCount.Store(int64(status.Pending))
				f.queuedCount.Store(int64(status.Queued))
				f.pendingExact.Store(true)
			}
			cancel()

			received := f.stats.received.Load()
			txPerSec := float64(received-lastReceived) / now.Sub(lastTick).Seconds()
			lastReceived = received
			lastTick = now

			p10, p25, p50, p75, p90 := f.tipPercentiles()
			f.mu.RLock()
			base := f.baseFee
			f.mu.RUnlock()

			snap := &models.MempoolSnapshot{
				At:           now,
				PendingCount: int(f.pendingCount.Load()),
				QueuedCount:  int(f.queuedCount.Load()),
				TrackedCount: tracked,
				BaseFeeGwei:  base,
				TipP10:       p10,
				TipP25:       p25,
				TipP50:       p50,
				TipP75:       p75,
				TipP90:       p90,
				TxPerSec:     txPerSec,
			}
			f.st.Enqueue(func(db *gorm.DB) error {
				return db.Create(snap).Error
			})

			f.hub.Publish(wire.StatsEvent(wire.StatsMsg{
				At:           now.UnixMilli(),
				Pending:      snap.PendingCount,
				Queued:       snap.QueuedCount,
				PendingExact: f.pendingExact.Load(),
				BaseFeeGwei:  base,
				TipP10:       p10,
				TipP50:       p50,
				TipP90:       p90,
				TxPerSec:     txPerSec,
			}))
		}
	}
}
