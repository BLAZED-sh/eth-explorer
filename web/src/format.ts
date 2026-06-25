export function shortHash(h: string, head = 8, tail = 6): string {
  if (!h) return "";
  const s = h.startsWith("0x") ? h.slice(2) : h;
  if (s.length <= head + tail) return "0x" + s;
  return `0x${s.slice(0, head)}…${s.slice(-tail)}`;
}

export function shortAddr(a: string, head = 6, tail = 4): string {
  return shortHash(a, head, tail);
}

export function fmtGwei(v: number): string {
  if (v === 0) return "0";
  if (v < 0.01) return "<0.01";
  if (v < 10) return v.toFixed(2);
  if (v < 100) return v.toFixed(1);
  return Math.round(v).toLocaleString();
}

export function fmtEth(v: number): string {
  if (v === 0) return "0";
  if (v < 0.0001) return "<0.0001";
  if (v < 1) return v.toFixed(4);
  if (v < 1000) return v.toFixed(3);
  return v.toLocaleString(undefined, { maximumFractionDigits: 1 });
}

export function fmtNum(n: number): string {
  return n.toLocaleString();
}

export function fmtBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export function fmtAge(unixMs: number, now = Date.now()): string {
  let s = Math.max(0, Math.floor((now - unixMs) / 1000));
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m ${s % 60}s`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
  return `${Math.floor(s / 86400)}d ${Math.floor((s % 86400) / 3600)}h`;
}

export function fmtDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`;
}

export function fmtTime(unix: number): string {
  return new Date(unix * 1000).toLocaleString();
}

export function txTypeLabel(t: number): string {
  switch (t) {
    case 0: return "Legacy";
    case 1: return "Access List";
    case 2: return "EIP-1559";
    case 3: return "Blob";
    case 4: return "Set Code";
    default: return `Type ${t}`;
  }
}

// Well-known 4-byte selectors for at-a-glance labels in the feed.
const METHODS: Record<string, string> = {
  "0xa9059cbb": "transfer",
  "0x23b872dd": "transferFrom",
  "0x095ea7b3": "approve",
  "0xa22cb465": "setApprovalForAll",
  "0x42842e0e": "safeTransferFrom",
  "0xd0e30db0": "deposit",
  "0x2e1a7d4d": "withdraw",
  "0x38ed1739": "swap",
  "0x7ff36ab5": "swap",
  "0x18cbafe5": "swap",
  "0x5c11d795": "swap",
  "0x3593564c": "swap",
  "0x04e45aaf": "swap",
  "0x5023b4df": "swap",
  "0xac9650d8": "multicall",
  "0x5ae401dc": "multicall",
  "0x1249c58b": "mint",
  "0xa0712d68": "mint",
  "0x6a761202": "execTransaction",
  "0xb6f9de95": "swap",
  "0x22895118": "stake",
};

export function methodLabel(sig: string, dataSize: number): string {
  if (!sig && dataSize === 0) return "transfer";
  return METHODS[sig] ?? sig;
}
