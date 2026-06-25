<script setup lang="ts">
// A calm, data-true mempool visual. Each pending tx is a small dot that
// enters from the left and drifts gently rightward toward a soft "block gate".
// Dot color = gas-tip heat (dim teal → bright cyan), size = ETH value. When a
// block lands, the txs it mined brighten and are collected at the gate, so the
// motion narrates pending → mined. 2D canvas only — no WebGL, no shaders.
import { onBeforeUnmount, onMounted, ref } from "vue";
import { onBlock, onPendingBatch, onStats } from "../store";
import type { TxLite } from "../types";

const canvasEl = ref<HTMLCanvasElement | null>(null);
const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

let teardown = () => {};
onBeforeUnmount(() => teardown());

interface Dot {
  x: number;
  y: number;
  r: number;
  vx: number;
  rank: number; // tip heat 0..1
  phase: number;
  freq: number;
  amp: number;
  born: number; // ms timestamp for fade-in
  mined: boolean;
  minedAt: number;
  hash: string;
}

// color ramp: dim teal → accent cyan → near-white cyan
function heat(rank: number): [number, number, number] {
  const t = rank < 0 ? 0 : rank > 1 ? 1 : rank;
  const lerp = (a: number, b: number, k: number) => Math.round(a + (b - a) * k);
  if (t < 0.55) {
    const k = t / 0.55;
    return [lerp(38, 44, k), lerp(110, 209, k), lerp(110, 209, k)];
  }
  const k = (t - 0.55) / 0.45;
  return [lerp(44, 198, k), lerp(209, 249, k), lerp(209, 242, k)];
}

function valueRadius(v: number): number {
  if (v > 10) return 3.2;
  if (v > 1) return 2.6;
  if (v > 0.01) return 2.1;
  return 1.7;
}

onMounted(() => {
  const canvas = canvasEl.value!;
  const host = canvas.parentElement ?? canvas;
  const ctx = canvas.getContext("2d")!;
  const dpr = Math.min(window.devicePixelRatio || 1, 2);
  let w = 0;
  let h = 0;
  // Block-gate gradient, cached across frames (geometry depends only on width).
  // The per-frame pulse is applied via globalAlpha, not by rebuilding it.
  let gateGrad: CanvasGradient | null = null;

  function resize() {
    const rect = host.getBoundingClientRect();
    w = Math.max(1, rect.width);
    h = Math.max(1, rect.height);
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(h * dpr);
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    gateGrad = ctx.createLinearGradient(w * 0.9 - 40, 0, w, 0);
    gateGrad.addColorStop(0, "rgba(54,226,226,0)");
    gateGrad.addColorStop(1, "rgba(54,226,226,1)");
    if (reduced) drawStatic();
  }

  const ro = new ResizeObserver(resize);
  ro.observe(host);
  resize();

  // ── reduced motion: one calm static field, no loop, no live churn ───────
  if (reduced) {
    teardown = () => ro.disconnect();
    return;
  }

  const MAX = 220;
  const CROSS_SECONDS = 13; // gentle: full width crossing
  const dots: Dot[] = [];
  const byHash = new Map<string, Dot>();
  let medianTip = 1.5; // gwei, refined by live stats
  let gatePulse = 0; // 0..1, flashes when a block lands
  let last = 0;
  let raf = 0;

  function spawn(tx: TxLite, now: number) {
    if (dots.length >= MAX || byHash.has(tx.hash)) return;
    const rank = tx.tipGwei > 0 ? tx.tipGwei / (tx.tipGwei + medianTip) : 0.05;
    const r = valueRadius(tx.valueEth);
    const d: Dot = {
      x: -r,
      y: 12 + Math.random() * Math.max(1, h - 24),
      r,
      vx: (w + 2 * r) / CROSS_SECONDS,
      rank,
      phase: Math.random() * Math.PI * 2,
      freq: 0.3 + Math.random() * 0.5,
      amp: 4 + Math.random() * 8,
      born: now,
      mined: false,
      minedAt: 0,
      hash: tx.hash,
    };
    dots.push(d);
    byHash.set(tx.hash, d);
  }

  function remove(i: number) {
    const d = dots[i];
    byHash.delete(d.hash);
    dots[i] = dots[dots.length - 1];
    dots.pop();
  }

  const offPending = onPendingBatch((txs) => {
    if (document.hidden) return;
    const now = performance.now();
    for (const tx of txs) spawn(tx, now);
  });

  const offBlock = onBlock((_block, minedHashes) => {
    gatePulse = 1;
    const now = performance.now();
    for (const hash of minedHashes) {
      const d = byHash.get(hash);
      if (d && !d.mined) {
        d.mined = true;
        d.minedAt = now;
      }
    }
  });

  const offStats = onStats((s) => {
    if (s.tipP50 > 0) medianTip = s.tipP50;
  });

  const gateX = () => w * 0.9;

  function tick(now: number) {
    const dt = last ? Math.min(0.05, (now - last) / 1000) : 0.016;
    last = now;
    gatePulse = Math.max(0, gatePulse - dt * 1.6);

    ctx.clearRect(0, 0, w, h);

    // soft block gate on the right — cached gradient, pulse via globalAlpha
    const gx = gateX();
    if (gateGrad) {
      ctx.save();
      ctx.globalAlpha = 0.05 + gatePulse * 0.22;
      ctx.fillStyle = gateGrad;
      ctx.fillRect(gx - 40, 0, w - (gx - 40), h);
      ctx.restore();
    }

    for (let i = dots.length - 1; i >= 0; i--) {
      const d = dots[i];
      let alpha: number;
      let glow = 1;

      if (d.mined) {
        const k = (now - d.minedAt) / 600; // collect + fade over 0.6s
        if (k >= 1) {
          remove(i);
          continue;
        }
        d.x += (gx - d.x) * Math.min(1, dt * 6);
        alpha = (1 - k) * 0.95;
        glow = 1.6 + k * 1.2; // brief brighten on inclusion
      } else {
        d.x += d.vx * dt;
        if (d.x > w + d.r) {
          remove(i);
          continue;
        }
        const age = (now - d.born) / 1000;
        const fadeIn = Math.min(1, age / 0.5);
        const nearEnd = Math.max(0, 1 - (d.x - gx) / (w - gx)); // dim as it passes the gate
        alpha = 0.85 * fadeIn * (d.x > gx ? nearEnd : 1);
      }

      const y = d.y + Math.sin(now / 1000 * d.freq + d.phase) * d.amp;
      const [cr, cg, cb] = heat(d.rank);

      // faint halo + bright core
      ctx.beginPath();
      ctx.arc(d.x, y, d.r * 2.6, 0, Math.PI * 2);
      ctx.fillStyle = `rgba(${cr},${cg},${cb},${alpha * 0.14 * glow})`;
      ctx.fill();

      ctx.beginPath();
      ctx.arc(d.x, y, d.r, 0, Math.PI * 2);
      ctx.fillStyle = `rgba(${cr},${cg},${cb},${Math.min(1, alpha * glow)})`;
      ctx.fill();
    }

    raf = requestAnimationFrame(tick);
  }
  raf = requestAnimationFrame(tick);

  function onVisibility() {
    if (document.hidden) {
      cancelAnimationFrame(raf);
      raf = 0;
      last = 0;
    } else if (!raf) {
      raf = requestAnimationFrame(tick);
    }
  }
  document.addEventListener("visibilitychange", onVisibility);

  teardown = () => {
    cancelAnimationFrame(raf);
    ro.disconnect();
    document.removeEventListener("visibilitychange", onVisibility);
    offPending();
    offBlock();
    offStats();
  };

  // ── static frame for reduced-motion ─────────────────────────────────────
  function drawStatic() {
    ctx.clearRect(0, 0, w, h);
    let seed = 1337;
    const rnd = () => {
      seed = (seed * 1103515245 + 12345) & 0x7fffffff;
      return seed / 0x7fffffff;
    };
    for (let i = 0; i < 120; i++) {
      const rank = rnd();
      const [cr, cg, cb] = heat(rank);
      const r = 1.5 + rnd() * 1.8;
      ctx.beginPath();
      ctx.arc(rnd() * w, rnd() * h, r, 0, Math.PI * 2);
      ctx.fillStyle = `rgba(${cr},${cg},${cb},${0.25 + rank * 0.35})`;
      ctx.fill();
    }
  }
});
</script>

<template>
  <div class="absolute inset-0" aria-hidden="true">
    <canvas ref="canvasEl" class="h-full w-full" />
  </div>
</template>
