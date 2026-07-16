import { watchEffect } from "vue";

export const SITE_ORIGIN = "https://explorer.blazed.sh";
export const DEFAULT_TITLE =
  "Blazed Explorer · Live Ethereum Block & Mempool Explorer";
export const DEFAULT_DESCRIPTION =
  "Explore Ethereum in real time: live mempool with pending transactions, blocks, gas prices, transaction and address details, plus contract decompilation. Powered by BLAZED.sh dedicated Ethereum nodes.";

const DEFAULT_ROBOTS = "index, follow, max-image-preview:large";

export interface PageMeta {
  title: string;
  description: string;
  /** Path for canonical + og:url + twitter:url, e.g. "/block/123".
   *  null removes the canonical tag (use on error/404 states). */
  canonicalPath?: string | null;
  /** Robots override, e.g. "noindex". Omit for the site default. */
  robots?: string;
}

function ensure(
  selector: string,
  create: () => HTMLElement,
): HTMLElement {
  let el = document.head.querySelector<HTMLElement>(selector);
  if (!el) {
    el = create();
    document.head.appendChild(el);
  }
  return el;
}

function setNamedMeta(attr: "name" | "property", key: string, content: string) {
  const el = ensure(`meta[${attr}="${key}"]`, () => {
    const m = document.createElement("meta");
    m.setAttribute(attr, key);
    return m;
  });
  el.setAttribute("content", content);
}

function apply(m: PageMeta) {
  document.title = m.title;
  setNamedMeta("name", "description", m.description);
  setNamedMeta("name", "robots", m.robots ?? DEFAULT_ROBOTS);

  const canonical = document.head.querySelector('link[rel="canonical"]');
  if (m.canonicalPath === null) {
    canonical?.remove();
  } else if (m.canonicalPath !== undefined) {
    const url = SITE_ORIGIN + m.canonicalPath;
    const el = ensure('link[rel="canonical"]', () => {
      const l = document.createElement("link");
      l.setAttribute("rel", "canonical");
      return l;
    });
    el.setAttribute("href", url);
    setNamedMeta("property", "og:url", url);
    setNamedMeta("name", "twitter:url", url);
  }

  setNamedMeta("property", "og:title", m.title);
  setNamedMeta("property", "og:description", m.description);
  setNamedMeta("name", "twitter:title", m.title);
  setNamedMeta("name", "twitter:description", m.description);
}

/**
 * Sets document title, description, canonical, robots, and OG/Twitter tags
 * for the calling view. Pass a getter to re-apply automatically when the
 * underlying refs change (e.g. once fetched data arrives). Views own their
 * meta exclusively per route, so nothing is restored on unmount.
 */
export function usePageMeta(meta: PageMeta | (() => PageMeta)) {
  watchEffect(() => {
    apply(typeof meta === "function" ? meta() : meta);
  });
}
