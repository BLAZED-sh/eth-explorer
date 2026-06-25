import type {
  BlockMsg, ContractCode, GasNow, HistoryPoint, SearchResult, TxDetail, TxLite,
} from "./types";

async function get<T>(path: string): Promise<T> {
  const res = await fetch(path);
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error ?? `HTTP ${res.status}`);
  }
  return res.json();
}

export interface TxPage {
  txs: TxLite[];
  nextCursor: number | null;
}

export const api = {
  txs(status = "pending", limit = 50, before?: number): Promise<TxPage> {
    const params = new URLSearchParams({ status, limit: String(limit) });
    if (before) params.set("before", String(before));
    return get(`/api/txs?${params}`);
  },
  tx(hash: string): Promise<TxDetail> {
    return get(`/api/txs/${hash}`);
  },
  address(addr: string, limit = 50, before?: number): Promise<TxPage & { address: string }> {
    const params = new URLSearchParams({ limit: String(limit) });
    if (before) params.set("before", String(before));
    return get(`/api/address/${addr}?${params}`);
  },
  code(addr: string): Promise<ContractCode> {
    return get(`/api/address/${addr}/code`);
  },
  blocks(limit = 25, before?: number): Promise<{ blocks: BlockMsg[] }> {
    const params = new URLSearchParams({ limit: String(limit) });
    if (before) params.set("before", String(before));
    return get(`/api/blocks?${params}`);
  },
  block(num: number | string): Promise<{ block: BlockMsg; txs: TxLite[] }> {
    return get(`/api/blocks/${num}`);
  },
  gas(): Promise<GasNow> {
    return get("/api/gas");
  },
  history(window: "1h" | "6h" | "24h" | "7d"): Promise<{ points: HistoryPoint[] }> {
    return get(`/api/stats/history?window=${window}`);
  },
  search(q: string): Promise<SearchResult> {
    return get(`/api/search?q=${encodeURIComponent(q)}`);
  },
};
