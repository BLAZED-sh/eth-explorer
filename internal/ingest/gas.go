package ingest

import (
	"sort"
	"time"

	"github.com/fipso/blazed-explorer/internal/wire"
)

const gasCacheTTL = 500 * time.Millisecond

// tipPercentiles computes priority-fee percentiles (gwei) over the in-memory
// pending pool.
func (f *Feed) tipPercentiles() (p10, p25, p50, p75, p90 float64) {
	f.mu.RLock()
	tips := make([]float64, 0, len(f.pool))
	for _, p := range f.pool {
		tips = append(tips, p.lite.TipGwei)
	}
	f.mu.RUnlock()
	if len(tips) == 0 {
		return 0, 0, 0, 0, 0
	}
	sort.Float64s(tips)
	pct := func(q float64) float64 {
		i := int(q * float64(len(tips)-1))
		return tips[i]
	}
	return pct(0.10), pct(0.25), pct(0.50), pct(0.75), pct(0.90)
}

// GasNow returns the current gas tiers, memoized for gasCacheTTL so a burst of
// /api/gas requests can't each trigger a full pool copy+sort under the lock.
func (f *Feed) GasNow() wire.GasNow {
	f.gasMu.Lock()
	defer f.gasMu.Unlock()
	if f.gasAt.IsZero() || time.Since(f.gasAt) >= gasCacheTTL {
		f.gasCache = f.computeGasNow()
		f.gasAt = time.Now()
	}
	return f.gasCache
}

// invalidateGas forces the next GasNow to recompute. Called on a new block so
// the block event carries fresh base fee + tiers. Must NOT be called while
// holding f.mu (lock-ordering: GasNow takes gasMu then f.mu).
func (f *Feed) invalidateGas() {
	f.gasMu.Lock()
	f.gasAt = time.Time{}
	f.gasMu.Unlock()
}

// computeGasNow derives slow/standard/fast/rapid prices from the current base
// fee plus pending-pool tip percentiles.
func (f *Feed) computeGasNow() wire.GasNow {
	_, p25, p50, p75, p90 := f.tipPercentiles()
	f.mu.RLock()
	base := f.baseFee
	tracked := len(f.pool)
	f.mu.RUnlock()
	// Before the first txpool_status poll (or on nodes without the txpool
	// namespace) fall back to the txs we track ourselves.
	pending := int(f.pendingCount.Load())
	if pending == 0 {
		pending = tracked
	}
	return wire.GasNow{
		BaseFeeGwei:  base,
		Slow:         base + p25,
		Standard:     base + p50,
		Fast:         base + p75,
		Rapid:        base + p90,
		PendingCount: pending,
		QueuedCount:  int(f.queuedCount.Load()),
		PendingExact: f.pendingExact.Load(),
	}
}
