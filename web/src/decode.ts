// Pure-TS decoding utilities for the transaction breakdown — no ABI library.
// All arithmetic stays in bigint; floats only appear at the display boundary.

import type { RawLog, RawReceipt, TxFull } from "./types";

// ─── numeric primitives ──────────────────────────────────────────────────────

export function hexToBigInt(hex?: string | null): bigint {
  if (!hex || hex === "0x") return 0n;
  try {
    return BigInt(hex);
  } catch {
    return 0n;
  }
}

/** Exact decimal string for v / 10^decimals — no float loss. */
export function formatUnits(v: bigint, decimals: number): string {
  const neg = v < 0n;
  if (neg) v = -v;
  const base = 10n ** BigInt(decimals);
  const int = v / base;
  const frac = (v % base).toString().padStart(decimals, "0").replace(/0+$/, "");
  const s = frac ? `${int}.${frac}` : int.toString();
  return neg ? `-${s}` : s;
}

export const formatGweiExact = (v: bigint) => formatUnits(v, 9);
export const formatEthExact = (v: bigint) => formatUnits(v, 18);

/** Backend serves gwei as float64; safe to round into wei below 2^53. */
export function gweiToWei(gwei: number): bigint {
  return BigInt(Math.round(gwei * 1e9));
}

/** Trim an exact decimal string for display (keep `sig` significant decimals). */
export function trimDecimals(s: string, sig = 6): string {
  const [int, frac] = s.split(".");
  if (!frac) return s;
  if (int !== "0") return frac.length > 4 ? `${int}.${frac.slice(0, 4)}` : s;
  // leading-zero fractions: keep first `sig` digits after the zeros run out
  const m = frac.match(/^(0*)(\d+)$/);
  if (!m) return s;
  const kept = m[1] + m[2].slice(0, sig);
  return `0.${kept}`;
}

// ─── calldata decoder (static types only) ────────────────────────────────────

export interface AbiParam {
  name: string;
  type: "address" | "uint256";
}

interface KnownMethod {
  name: string;
  params: AbiParam[];
}

const SELECTORS: Record<string, KnownMethod> = {
  "0xa9059cbb": { name: "transfer", params: [{ name: "to", type: "address" }, { name: "amount", type: "uint256" }] },
  "0x095ea7b3": { name: "approve", params: [{ name: "spender", type: "address" }, { name: "amount", type: "uint256" }] },
  "0x23b872dd": { name: "transferFrom", params: [{ name: "from", type: "address" }, { name: "to", type: "address" }, { name: "amount", type: "uint256" }] },
  "0x42842e0e": { name: "safeTransferFrom", params: [{ name: "from", type: "address" }, { name: "to", type: "address" }, { name: "tokenId", type: "uint256" }] },
  "0xd0e30db0": { name: "deposit", params: [] },
  "0x2e1a7d4d": { name: "withdraw", params: [{ name: "wad", type: "uint256" }] },
};

export interface DecodedParam {
  name: string;
  type: string;
  value: string; // address hex or bigint decimal string
  raw: string;   // the full 32-byte word
}

export interface DecodedCall {
  name: string;
  params: DecodedParam[];
}

export function decodeCalldata(input: string): DecodedCall | null {
  if (!input || input.length < 10) return null;
  const method = SELECTORS[input.slice(0, 10).toLowerCase()];
  if (!method) return null;
  const body = input.slice(10);
  if (body.length < method.params.length * 64) return null; // malformed → hex viewer
  const params: DecodedParam[] = method.params.map((p, i) => {
    const word = body.slice(i * 64, (i + 1) * 64);
    return {
      name: p.name,
      type: p.type,
      value: p.type === "address" ? "0x" + word.slice(24) : hexToBigInt("0x" + word).toString(),
      raw: "0x" + word,
    };
  });
  return { name: method.name, params };
}

export interface HexWord {
  offset: number; // bytes
  hex: string;    // 64 hex chars (last word may be shorter)
}

export function toWords(input: string): { selector: string | null; words: HexWord[] } {
  if (!input || input === "0x") return { selector: null, words: [] };
  const raw = input.startsWith("0x") ? input.slice(2) : input;
  // calldata with a selector is 4 bytes + 32-byte words; bare data is just words
  const hasSelector = raw.length % 64 === 8;
  const selector = hasSelector ? "0x" + raw.slice(0, 8) : null;
  const body = hasSelector ? raw.slice(8) : raw;
  const words: HexWord[] = [];
  for (let i = 0; i < body.length; i += 64) {
    words.push({ offset: i / 2, hex: body.slice(i, i + 64) });
  }
  return { selector, words };
}

// ─── receipt log decoder ─────────────────────────────────────────────────────

const TOPIC_TRANSFER = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef";
const TOPIC_APPROVAL = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925";
const TOPIC_WETH_DEPOSIT = "0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c";
const TOPIC_WETH_WITHDRAWAL = "0x7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65";

const topicAddr = (t: string) => "0x" + t.slice(26);

export type DecodedLog =
  | { kind: "erc20-transfer"; token: string; from: string; to: string; value: bigint }
  | { kind: "erc721-transfer"; token: string; from: string; to: string; tokenId: bigint }
  | { kind: "approval"; token: string; owner: string; spender: string; value: bigint; isNft: boolean }
  | { kind: "weth-deposit" | "weth-withdrawal"; token: string; account: string; value: bigint }
  | { kind: "unknown"; address: string; topic0: string | null; dataBytes: number };

export function decodeLogs(logs: RawLog[]): DecodedLog[] {
  return logs.map((log): DecodedLog => {
    const t0 = log.topics[0]?.toLowerCase() ?? null;
    const n = log.topics.length;
    switch (t0) {
      case TOPIC_TRANSFER:
        // ERC-20: value in data, 3 topics. ERC-721: tokenId indexed, 4 topics.
        if (n === 3) {
          return { kind: "erc20-transfer", token: log.address, from: topicAddr(log.topics[1]), to: topicAddr(log.topics[2]), value: hexToBigInt(log.data) };
        }
        if (n === 4) {
          return { kind: "erc721-transfer", token: log.address, from: topicAddr(log.topics[1]), to: topicAddr(log.topics[2]), tokenId: hexToBigInt(log.topics[3]) };
        }
        break;
      case TOPIC_APPROVAL:
        if (n === 3) {
          return { kind: "approval", token: log.address, owner: topicAddr(log.topics[1]), spender: topicAddr(log.topics[2]), value: hexToBigInt(log.data), isNft: false };
        }
        if (n === 4) {
          return { kind: "approval", token: log.address, owner: topicAddr(log.topics[1]), spender: topicAddr(log.topics[2]), value: hexToBigInt(log.topics[3]), isNft: true };
        }
        break;
      case TOPIC_WETH_DEPOSIT:
        if (n === 2) {
          return { kind: "weth-deposit", token: log.address, account: topicAddr(log.topics[1]), value: hexToBigInt(log.data) };
        }
        break;
      case TOPIC_WETH_WITHDRAWAL:
        if (n === 2) {
          return { kind: "weth-withdrawal", token: log.address, account: topicAddr(log.topics[1]), value: hexToBigInt(log.data) };
        }
        break;
    }
    return { kind: "unknown", address: log.address, topic0: t0, dataBytes: log.data && log.data !== "0x" ? (log.data.length - 2) / 2 : 0 };
  });
}

// ─── fee breakdown ───────────────────────────────────────────────────────────

export interface FeeBreakdown {
  gasUsed: bigint;
  gasLimit: bigint;
  effectiveGasPriceWei: bigint;
  totalFeeWei: bigint;
  baseFeeWei: bigint | null;
  burnedWei: bigint | null;
  tipWei: bigint | null;
  maxCostWei: bigint | null; // type >= 2 only
  savedWei: bigint | null;
  baseFeeSource: "block" | "derived" | null;
  approximate: boolean; // effectiveGasPrice missing → reconstructed from tx fields
  blobFeeWei: bigint | null;
}

export function computeFees(tx: TxFull, receipt: RawReceipt, blockBaseFeeGwei: number | null): FeeBreakdown {
  const gasUsed = hexToBigInt(receipt.gasUsed);
  const gasLimit = BigInt(tx.gasLimit);

  let approximate = false;
  let effectiveGasPriceWei = hexToBigInt(receipt.effectiveGasPrice);
  if (effectiveGasPriceWei === 0n) {
    effectiveGasPriceWei = gweiToWei(tx.gasPriceGwei);
    approximate = true;
  }
  const totalFeeWei = gasUsed * effectiveGasPriceWei;

  let baseFeeWei: bigint | null = null;
  let baseFeeSource: FeeBreakdown["baseFeeSource"] = null;
  if (blockBaseFeeGwei != null && blockBaseFeeGwei > 0) {
    baseFeeWei = gweiToWei(blockBaseFeeGwei);
    baseFeeSource = "block";
  } else if (tx.tipGwei > 0) {
    // Only exact when the priority fee wasn't clamped by maxFee.
    const derived = effectiveGasPriceWei - gweiToWei(tx.tipGwei);
    if (derived > 0n) {
      baseFeeWei = derived;
      baseFeeSource = "derived";
    }
  }

  let burnedWei: bigint | null = null;
  let tipWei: bigint | null = null;
  if (baseFeeWei != null) {
    burnedWei = gasUsed * (baseFeeWei < effectiveGasPriceWei ? baseFeeWei : effectiveGasPriceWei);
    tipWei = totalFeeWei - burnedWei;
  }

  let maxCostWei: bigint | null = null;
  let savedWei: bigint | null = null;
  if (tx.type >= 2 && tx.maxFeeGwei > 0) {
    maxCostWei = gasUsed * gweiToWei(tx.maxFeeGwei);
    savedWei = maxCostWei - totalFeeWei;
    if (savedWei < 0n) savedWei = 0n;
  }

  let blobFeeWei: bigint | null = null;
  if (receipt.blobGasUsed && receipt.blobGasPrice) {
    blobFeeWei = hexToBigInt(receipt.blobGasUsed) * hexToBigInt(receipt.blobGasPrice);
  }

  return {
    gasUsed, gasLimit, effectiveGasPriceWei, totalFeeWei,
    baseFeeWei, burnedWei, tipWei, maxCostWei, savedWei,
    baseFeeSource, approximate, blobFeeWei,
  };
}

/** bigint-safe percentage of part/whole, 2 decimals. */
export function pct(part: bigint, whole: bigint): number {
  if (whole === 0n) return 0;
  return Number((part * 10000n) / whole) / 100;
}

// ─── lifecycle ───────────────────────────────────────────────────────────────

export interface LifecycleStep {
  key: "seen" | "pending" | "mined" | "failed" | "dropped" | "replaced";
  state: "done" | "active" | "future" | "terminal-ok" | "terminal-bad" | "terminal-warn";
  label: string;
  detail: string;
}

export function lifecycle(tx: TxFull, receipt: RawReceipt | null, nowMs: number): LifecycleStep[] {
  const seenMs = Date.parse(tx.firstSeen);
  const steps: LifecycleStep[] = [
    {
      key: "seen",
      state: "done",
      label: tx.seenInMempool ? "Seen in mempool" : "Seen in block",
      detail: new Date(seenMs).toLocaleTimeString(),
    },
  ];

  switch (tx.status) {
    case "pending": {
      const elapsed = Math.max(0, Math.floor((nowMs - seenMs) / 1000));
      steps.push({ key: "pending", state: "active", label: "Pending", detail: `${elapsed}s and counting` });
      steps.push({ key: "mined", state: "future", label: "Inclusion", detail: "awaiting block" });
      break;
    }
    case "mined": {
      const failed = receipt?.status === "0x0";
      const dur = tx.confirmMs != null ? `${(tx.confirmMs / 1000).toFixed(1)}s in pool` : "";
      steps.push({ key: "pending", state: "done", label: "Pending", detail: dur });
      steps.push({
        key: failed ? "failed" : "mined",
        state: failed ? "terminal-bad" : "terminal-ok",
        label: failed ? "Reverted" : "Mined",
        detail: tx.blockNumber != null ? `block #${tx.blockNumber}` : "",
      });
      break;
    }
    case "replaced":
      steps.push({ key: "pending", state: "done", label: "Pending", detail: "" });
      steps.push({ key: "replaced", state: "terminal-warn", label: "Replaced", detail: `same nonce (${tx.nonce})` });
      break;
    case "dropped":
      steps.push({ key: "pending", state: "done", label: "Pending", detail: "" });
      steps.push({ key: "dropped", state: "terminal-bad", label: "Dropped", detail: "evicted from mempool" });
      break;
  }
  return steps;
}
