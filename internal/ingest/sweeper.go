package ingest

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/fipso/blazed-explorer/internal/models"
)

const poolEntryTTL = 10 * time.Minute

// runSweeper resolves stale pending txs: still pending on the node (leave),
// mined in a block we processed without it (mark mined), gone with a mined
// same-sender same-nonce sibling (replaced), or just gone (dropped). It also
// expires ancient entries from the in-memory pool so percentiles stay honest.
func (f *Feed) runSweeper(ctx context.Context) {
	t := time.NewTicker(sweepInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			f.expirePoolEntries()
			f.sweepOnce(ctx)
		}
	}
}

func (f *Feed) expirePoolEntries() {
	cutoff := time.Now().Add(-poolEntryTTL)
	n := 0
	f.mu.Lock()
	for h, p := range f.pool {
		if p.firstSeen.Before(cutoff) {
			delete(f.pool, h)
			n++
		}
	}
	f.mu.Unlock()
	if n > 0 {
		f.stats.evicted.Add(uint64(n))
	}
}

func (f *Feed) sweepOnce(ctx context.Context) {
	stale, err := f.st.StalePending(time.Now().Add(-sweepMinAge), sweepBatch)
	if err != nil {
		log.Warn().Err(err).Msg("sweeper query failed")
		return
	}
	if len(stale) == 0 {
		return
	}

	batch := make([]rpc.BatchElem, len(stale))
	results := make([]*rawTx, len(stale))
	for i, tx := range stale {
		batch[i] = rpc.BatchElem{
			Method: "eth_getTransactionByHash",
			Args:   []any{tx.Hash},
			Result: &results[i],
		}
	}
	rctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := f.httpRPC.BatchCallContext(rctx, batch); err != nil {
		log.Warn().Err(err).Msg("sweeper batch call failed")
		return
	}

	var minedN, droppedN, replacedN int
	for i, tx := range stale {
		if batch[i].Error != nil {
			continue
		}
		rt := results[i]
		switch {
		case rt == nil:
			// Gone from the node: replaced if a sibling with the same
			// sender+nonce mined, otherwise dropped.
			if sibling, err := f.st.MinedTxBySenderNonce(tx.Sender, tx.Nonce, tx.Hash); err == nil {
				replacedN++
				f.markReplaced(tx.Hash, sibling.Hash)
			} else {
				droppedN++
				f.markDropped(tx.Hash)
			}
		case rt.BlockNumber != nil:
			// Mined but we missed the head (or it was inserted before our
			// block processing ran). Backfill from our own block row if we
			// have it.
			minedN++
			f.markMined(tx, rt)
		}
		// Still pending on the node: leave it alone.
	}
	if minedN+droppedN+replacedN > 0 {
		log.Info().Int("mined", minedN).Int("dropped", droppedN).Int("replaced", replacedN).
			Int("checked", len(stale)).Msg("sweeper resolved stale txs")
	}
}

func (f *Feed) markDropped(hash string) {
	f.st.Enqueue(func(db *gorm.DB) error {
		return db.Model(&models.Tx{}).
			Where("hash = ? AND status = ?", hash, models.StatusPending).
			Update("status", models.StatusDropped).Error
	})
}

func (f *Feed) markReplaced(hash, byHash string) {
	f.st.Enqueue(func(db *gorm.DB) error {
		return db.Model(&models.Tx{}).
			Where("hash = ? AND status = ?", hash, models.StatusPending).
			Updates(map[string]any{
				"status":      models.StatusReplaced,
				"replaced_by": byHash,
			}).Error
	})
}

func (f *Feed) markMined(tx models.Tx, rt *rawTx) {
	blockNum := rt.BlockNumber.ToInt().Uint64()
	var position *int
	if rt.TransactionIndex != nil {
		p := int(*rt.TransactionIndex)
		position = &p
	}
	seen := tx.SeenInMempool
	firstSeen := tx.FirstSeen
	hash := strings.ToLower(tx.Hash)

	f.st.Enqueue(func(db *gorm.DB) error {
		updates := map[string]any{
			"status":       models.StatusMined,
			"block_number": blockNum,
		}
		if position != nil {
			updates["block_position"] = *position
		}
		// If we stored this block, backfill timing from it.
		var blk models.Block
		if err := db.Where("number = ?", blockNum).First(&blk).Error; err == nil {
			updates["mined_at"] = blk.ReceivedAt
			if seen {
				updates["confirm_ms"] = blk.ReceivedAt.Sub(firstSeen).Milliseconds()
			}
		}
		return db.Model(&models.Tx{}).Where("hash = ?", hash).Updates(updates).Error
	})
}
