import { shortAddr } from "./format";

// Curated builder / fee-recipient name tags (keys MUST be lowercase).
// These are the block fee-recipient (`miner`) addresses the dominant mainnet
// builders pay out to. They cover the large majority of blocks; anything not
// listed falls back to a short address, so correctness degrades gracefully.
// Cross-check / extend against etherscan name tags and relayscan.io.
const BUILDERS: Record<string, string> = {
  "0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5": "beaverbuild",
  "0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97": "Titan Builder",
  "0x1f9090aae28b8a3dceadf281b0f12828e676c326": "rsync-builder",
  "0xdafea492d9c6733ae3d56b7ed1adb60692c98bc5": "Flashbots",
  "0x690b9a9e9aa1c9db991c7721a92d351db4fac990": "builder0x69",
  "0xf2f5c73fa04406b1995e397b55c24ab1f3ea726c": "bloXroute: Max Profit",
  "0x4675c7e5baafbffbca748158becba61ef3b0a263": "bloXroute: Regulated",
  "0x199d5ed7f45f4ee35960cf22eade2076e95b253f": "bloXroute: Ethical",
  "0x3b64216ad1a58f61538b4fa1b27327675ab7ed67": "Manifold",
  "0xb646d87963da1fb9d192ddba775f24f33e857128": "rsync-builder",
  "0xe688b84b23f322a994a53dbf8e15fa82cdb71127": "Gambit Labs",
};

export interface Builder {
  name: string;
  /** true when the address resolves to a known builder tag */
  known: boolean;
}

export function builderName(addr: string | null | undefined): Builder {
  if (!addr) return { name: "unknown", known: false };
  const tag = BUILDERS[addr.toLowerCase()];
  return tag ? { name: tag, known: true } : { name: shortAddr(addr), known: false };
}
