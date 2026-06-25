import { reactive } from "vue";
import type { BlockMsg, GasNow, StatsMsg, TxLite } from "./types";
import { api } from "./api";

const FEED_CAP = 10;
const CONFIRMED_CAP = 10;
const BLOCK_CAP = 12;

type StatsListener = (s: StatsMsg) => void;
const statsListeners = new Set<StatsListener>();

// Raw listener bridges for the 3D scene: fire with plain pre-proxy objects,
// before any hidden/paused short-circuits, so consumers see the full stream.
type PendingBatchListener = (txs: TxLite[]) => void;
type BlockListener = (block: BlockMsg, minedHashes: string[]) => void;
const pendingBatchListeners = new Set<PendingBatchListener>();
const blockListeners = new Set<BlockListener>();

export function onPendingBatch(fn: PendingBatchListener): () => void {
  pendingBatchListeners.add(fn);
  return () => pendingBatchListeners.delete(fn);
}

export function onBlock(fn: BlockListener): () => void {
  blockListeners.add(fn);
  return () => blockListeners.delete(fn);
}

export const store = reactive({
  connected: false,
  chainId: 0,
  head: 0,
  gas: null as GasNow | null,
  stats: null as StatsMsg | null,
  feed: [] as TxLite[],
  /** latest txs we watched go pending → mined, newest first */
  confirmed: [] as TxLite[],
  blocks: [] as BlockMsg[],
  paused: false,
  pausedBuffer: [] as TxLite[],
  /** hashes recently mined — LiveTxFeed flashes + retires these rows */
  justMined: new Set<string>(),
});

export function onStats(fn: StatsListener): () => void {
  statsListeners.add(fn);
  return () => statsListeners.delete(fn);
}

export function applyHello(chainId: number, head: number, gas: GasNow, blocks: BlockMsg[], txs: TxLite[]) {
  store.chainId = chainId;
  store.head = head;
  store.gas = gas;
  store.blocks = blocks.slice(0, BLOCK_CAP);
  // hello txs arrive oldest-first; the feed renders newest-first
  store.feed = txs.slice().reverse().slice(0, FEED_CAP);
  store.pausedBuffer = [];
  store.justMined.clear();

  // Seed the Confirmed tab once so it isn't empty before the first block lands.
  // The live confirmed feed is then driven by applyBlock as we watch txs mine.
  api.txs("mined", CONFIRMED_CAP)
    .then((page) => {
      if (!store.confirmed.length) store.confirmed = page.txs.slice(0, CONFIRMED_CAP);
    })
    .catch(() => {});
}

export function applyPendingBatch(txs: TxLite[]) {
  pendingBatchListeners.forEach((fn) => fn(txs));
  if (document.hidden) return; // don't churn the DOM in background tabs
  if (store.paused) {
    store.pausedBuffer = [...txs.slice().reverse(), ...store.pausedBuffer].slice(0, FEED_CAP);
    return;
  }
  store.feed = [...txs.slice().reverse(), ...store.feed].slice(0, FEED_CAP);
}

export function applyBlock(block: BlockMsg, minedHashes: string[], gas: GasNow) {
  blockListeners.forEach((fn) => fn(block, minedHashes));
  store.head = block.number;
  store.gas = gas;
  store.blocks = [block, ...store.blocks.filter((b) => b.number !== block.number)].slice(0, BLOCK_CAP);

  const mined = new Set(minedHashes);
  const newlyConfirmed: TxLite[] = [];
  let touched = false;
  for (const tx of store.feed) {
    if (mined.has(tx.hash) && tx.status === "pending") {
      tx.status = "mined";
      store.justMined.add(tx.hash);
      newlyConfirmed.push({ ...tx });
      touched = true;
    }
  }
  if (newlyConfirmed.length) {
    // newest first; de-dupe against what's already shown
    const seen = new Set(newlyConfirmed.map((t) => t.hash));
    store.confirmed = [
      ...newlyConfirmed,
      ...store.confirmed.filter((t) => !seen.has(t.hash)),
    ].slice(0, CONFIRMED_CAP);
  }
  if (touched) {
    // Retire flashed rows after their green-out animation.
    setTimeout(() => {
      store.feed = store.feed.filter((tx) => !store.justMined.has(tx.hash));
      store.justMined.clear();
    }, 1500);
  }
}

export function applyStats(s: StatsMsg) {
  store.stats = s;
  if (store.gas) {
    store.gas.pendingCount = s.pending;
    store.gas.queuedCount = s.queued;
    store.gas.pendingExact = s.pendingExact;
  }
  statsListeners.forEach((fn) => fn(s));
}

export function setPaused(p: boolean) {
  store.paused = p;
  if (!p && store.pausedBuffer.length) {
    store.feed = [...store.pausedBuffer, ...store.feed].slice(0, FEED_CAP);
    store.pausedBuffer = [];
  }
}
