package ingest

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"

	"github.com/fipso/blazed-explorer/internal/wire"
)

// runPending subscribes with the second-arg `true` flag ("include the full tx
// body in each notification"), avoiding a per-tx eth_getTransactionByHash
// round-trip and the race against block production.
func (f *Feed) runPending(ctx context.Context) {
	for ctx.Err() == nil {
		ch := make(chan *rawTx, 1024)
		sub, err := f.wsRPC.EthSubscribe(ctx, ch, "newPendingTransactions", true)
		if err != nil {
			log.Error().Err(err).Msg("newPendingTransactions(full) subscribe failed; retrying")
			sleepCtx(ctx, 2*time.Second)
			continue
		}
		log.Info().Msg("subscribed to newPendingTransactions (full bodies)")
		err = f.consumePending(ctx, sub, ch)
		log.Warn().Err(err).Msg("pending subscription dropped; resubscribing")
		sleepCtx(ctx, time.Second)
	}
}

func (f *Feed) consumePending(ctx context.Context, sub *rpc.ClientSubscription, ch <-chan *rawTx) error {
	defer sub.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			return err
		case rt := <-ch:
			if rt == nil {
				continue
			}
			f.acceptPending(rt)
		}
	}
}

func (f *Feed) acceptPending(rt *rawTx) {
	// Defensive: skip anything the node tagged as already mined.
	if rt.BlockNumber != nil {
		return
	}
	now := time.Now()
	m := f.rawTxToModel(rt, now)
	lite := wire.TxLiteFromModel(m)

	f.mu.Lock()
	if _, exists := f.pool[rt.Hash]; exists {
		f.mu.Unlock()
		return
	}
	if len(f.pool) >= f.maxPool {
		// FIFO evict the oldest entry to make room.
		var evictH common.Hash
		var evictTime time.Time
		first := true
		for h, p := range f.pool {
			if first || p.firstSeen.Before(evictTime) {
				evictH = h
				evictTime = p.firstSeen
				first = false
			}
		}
		delete(f.pool, evictH)
		f.stats.evicted.Add(1)
	}
	f.pool[rt.Hash] = &poolTx{lite: lite, firstSeen: now}
	f.recent = append(f.recent, lite)
	if len(f.recent) > helloTxRing {
		f.recent = f.recent[len(f.recent)-helloTxRing:]
	}
	f.mu.Unlock()

	f.stats.received.Add(1)
	if !f.st.EnqueueTx(m) {
		f.stats.dropped.Add(1)
	}
	select {
	case f.pendingCh <- lite:
	default:
	}
}

// runBatcher flushes accepted pending txs to the browser hub in batches so a
// busy mainnet mempool doesn't generate hundreds of frames per second.
func (f *Feed) runBatcher(ctx context.Context) {
	tick := time.NewTicker(batchFlushEvery)
	defer tick.Stop()
	buf := make([]wire.TxLite, 0, batchMaxSize)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		f.hub.Publish(wire.PendingBatchEvent(buf))
		buf = buf[:0]
	}
	for {
		select {
		case <-ctx.Done():
			return
		case lite := <-f.pendingCh:
			buf = append(buf, lite)
			if len(buf) >= batchMaxSize {
				flush()
			}
		case <-tick.C:
			flush()
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
