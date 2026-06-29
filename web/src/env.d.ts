/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Absolute origin for REST API calls, e.g. "https://api.example.com". Empty = same-origin (default). */
  readonly VITE_API_BASE?: string;
  /** Full WebSocket URL, e.g. "wss://api.example.com/ws". Empty = derived from VITE_API_BASE or the page origin. */
  readonly VITE_WS_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
