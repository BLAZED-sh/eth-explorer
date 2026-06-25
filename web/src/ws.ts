import { applyBlock, applyHello, applyPendingBatch, applyStats, store } from "./store";
import type { WsEvent } from "./types";

let sock: WebSocket | null = null;
let backoff = 1000;
let reconnectTimer: number | undefined;

function wsURL(): string {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  return `${proto}://${location.host}/ws`;
}

export function connect() {
  clearTimeout(reconnectTimer);
  sock = new WebSocket(wsURL());

  sock.onopen = () => {
    store.connected = true;
    backoff = 1000;
  };

  sock.onmessage = (ev) => {
    let msg: WsEvent;
    try {
      msg = JSON.parse(ev.data);
    } catch {
      return;
    }
    switch (msg.t) {
      case "hello":
        applyHello(msg.chainId, msg.head, msg.gas, msg.recentBlocks ?? [], msg.txs ?? []);
        break;
      case "pending_batch":
        applyPendingBatch(msg.txs ?? []);
        break;
      case "block":
        applyBlock(msg.block, msg.minedHashes ?? [], msg.gas);
        break;
      case "stats":
        applyStats(msg);
        break;
    }
  };

  sock.onclose = () => {
    store.connected = false;
    scheduleReconnect();
  };
  sock.onerror = () => {
    sock?.close();
  };
}

function scheduleReconnect() {
  clearTimeout(reconnectTimer);
  reconnectTimer = window.setTimeout(connect, backoff);
  backoff = Math.min(backoff * 2, 30_000);
}

document.addEventListener("visibilitychange", () => {
  if (!document.hidden && sock?.readyState !== WebSocket.OPEN) {
    backoff = 1000;
    connect();
  }
});
