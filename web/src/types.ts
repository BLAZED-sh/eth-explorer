// Mirrors internal/wire/wire.go on the backend.

export interface TxLite {
  hash: string;
  from: string;
  to: string | null;
  valueEth: number;
  gasLimit: number;
  gasPriceGwei: number;
  tipGwei: number;
  type: number;
  dataSize: number;
  methodSig: string;
  status: string;
  firstSeen: number; // unix ms
}

export interface TxFull {
  hash: string;
  from: string;
  to: string | null;
  nonce: number;
  valueWei: string;
  valueEth: number;
  gasLimit: number;
  gasPriceGwei: number;
  maxFeeGwei: number;
  tipGwei: number;
  type: number;
  dataSize: number;
  methodSig: string;
  status: string;
  seenInMempool: boolean;
  firstSeen: string; // RFC3339
  blockNumber: number | null;
  blockPosition: number | null;
  minedAt: string | null;
  confirmMs: number | null;
  replacedBy: string | null;
}

// Raw eth_getTransactionReceipt JSON passed verbatim by the backend. Every
// field optional — geth/erigon/nethermind responses differ.
export interface RawLog {
  address: string;
  topics: string[];
  data: string;
  logIndex?: string;
  removed?: boolean;
}

export interface RawReceipt {
  status?: string; // "0x1" | "0x0"; pre-Byzantium gives `root` instead
  gasUsed?: string;
  effectiveGasPrice?: string; // wei hex; some clients omit it
  cumulativeGasUsed?: string;
  type?: string;
  logs?: RawLog[];
  contractAddress?: string | null;
  blobGasUsed?: string; // type-3 only
  blobGasPrice?: string;
  blockNumber?: string;
}

export interface TxDetail {
  tx: TxFull;
  input: string;
  receipt: RawReceipt | null;
}

// GET /api/address/{addr}/code — contract detection + runtime bytecode.
export interface ContractCode {
  address: string;
  isContract: boolean;
  // EIP-7702: set when the account is a delegated EOA (code is 0xef0100||addr).
  delegatedTo: string | null;
  bytecode: string; // "0x" when not a contract
  codeSize: number; // bytes
  // Present only when we ingested the deploy tx (most contracts predate our DB).
  creation: { txHash: string; deployer: string; blockNumber: number | null } | null;
}

export interface BlockMsg {
  number: number;
  hash: string;
  timestamp: number; // unix s
  txCount: number;
  knownCount: number;
  gasUsed: number;
  gasLimit: number;
  utilization: number;
  baseFeeGwei: number;
  miner: string;
}

export interface GasNow {
  baseFeeGwei: number;
  slow: number;
  standard: number;
  fast: number;
  rapid: number;
  pendingCount: number;
  queuedCount: number;
  pendingExact: boolean; // false when pendingCount is the tracked-pool floor, not node-exact
}

export interface StatsMsg {
  at: number; // unix ms
  pending: number;
  queued: number;
  pendingExact: boolean; // false when pending is the tracked-pool floor, not node-exact
  baseFeeGwei: number;
  tipP10: number;
  tipP50: number;
  tipP90: number;
  txPerSec: number;
}

export interface HistoryPoint {
  t: number; // unix ms
  pending: number;
  queued: number;
  baseFee: number;
  tipP10: number;
  tipP50: number;
  tipP90: number;
  txPerSec: number;
}

export type SearchResult = {
  type: "tx" | "address" | "block" | "none";
  ref: string;
};

// WS events
export type WsEvent =
  | ({ t: "hello"; chainId: number; head: number; gas: GasNow; recentBlocks: BlockMsg[]; txs: TxLite[] })
  | ({ t: "pending_batch"; txs: TxLite[] })
  | ({ t: "block"; block: BlockMsg; minedHashes: string[]; gas: GasNow })
  | ({ t: "stats" } & StatsMsg);
