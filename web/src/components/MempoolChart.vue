<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";
import { api } from "../api";
import { onStats } from "../store";
import type { HistoryPoint, StatsMsg } from "../types";
import { fmtGwei } from "../format";

const props = defineProps<{
  mode: "size" | "fees";
  title: string;
}>();

type Win = "1h" | "6h" | "24h";
const win = ref<Win>("1h");
const winSeconds: Record<Win, number> = { "1h": 3600, "6h": 6 * 3600, "24h": 24 * 3600 };

const el = ref<HTMLElement | null>(null);
const wrap = ref<HTMLElement | null>(null);
let u: uPlot | null = null;
let ro: ResizeObserver | null = null;
let offStats: (() => void) | null = null;

// column-major uPlot data
let data: number[][] = [];

const CYAN = "#2CD1D1";
const CYAN_DIM = "rgba(44, 209, 209, 0.5)";
const PURPLE = "#627EEA";
const GRID = "rgba(44, 209, 209, 0.07)";
const TICK = "rgba(255, 255, 255, 0.35)";

const axisFont = "10px 'JetBrains Mono', monospace";

function axis(extra: Partial<uPlot.Axis> = {}): uPlot.Axis {
  return {
    stroke: TICK,
    font: axisFont,
    grid: { stroke: GRID, width: 1 },
    ticks: { stroke: GRID, width: 1 },
    ...extra,
  };
}

function makeOpts(width: number): uPlot.Options {
  const common: Partial<uPlot.Options> = {
    width,
    height: 240,
    legend: { show: false },
    cursor: {
      points: { size: 5, fill: CYAN },
      drag: { x: false, y: false, setScale: false },
    },
    padding: [12, 8, 0, 0],
  };

  if (props.mode === "size") {
    return {
      ...common,
      series: [
        {},
        {
          label: "pending",
          stroke: CYAN,
          width: 1.5,
          fill: (self) => {
            const grad = self.ctx.createLinearGradient(0, 0, 0, self.bbox.height);
            grad.addColorStop(0, "rgba(44, 209, 209, 0.22)");
            grad.addColorStop(1, "rgba(44, 209, 209, 0)");
            return grad;
          },
        },
        { label: "queued", stroke: PURPLE, width: 1.2, dash: [4, 3] },
      ],
      axes: [
        axis(),
        axis({
          size: 56,
          values: (_, ticks) => ticks.map((v) => (v >= 1000 ? `${(v / 1000).toFixed(v >= 10000 ? 0 : 1)}k` : String(v))),
        }),
      ],
      scales: { y: { range: (_u, _min, max) => [0, max * 1.1 || 10] } },
    } as uPlot.Options;
  }

  return {
    ...common,
    series: [
      {},
      { label: "p90", stroke: "transparent" },
      { label: "p10", stroke: "transparent" },
      { label: "median", stroke: CYAN, width: 1.6 },
      { label: "base fee", stroke: PURPLE, width: 1.2, dash: [4, 3] },
    ],
    bands: [{ series: [1, 2], fill: "rgba(44, 209, 209, 0.10)" }],
    axes: [
      axis(),
      axis({ size: 56, values: (_, ticks) => ticks.map((v) => fmtGwei(v)) }),
    ],
    scales: { y: { range: (_u, min, max) => [Math.max(0, min * 0.9), max * 1.1 || 1] } },
  } as uPlot.Options;
}

function toData(points: HistoryPoint[]): number[][] {
  const ts = points.map((p) => p.t / 1000);
  if (props.mode === "size") {
    return [ts, points.map((p) => p.pending), points.map((p) => p.queued)];
  }
  return [
    ts,
    points.map((p) => p.baseFee + p.tipP90),
    points.map((p) => p.baseFee + p.tipP10),
    points.map((p) => p.baseFee + p.tipP50),
    points.map((p) => p.baseFee),
  ];
}

function appendStats(s: StatsMsg) {
  if (!u || data.length === 0) return;
  const t = s.at / 1000;
  data[0].push(t);
  if (props.mode === "size") {
    data[1].push(s.pending);
    data[2].push(s.queued);
  } else {
    data[1].push(s.baseFeeGwei + s.tipP90);
    data[2].push(s.baseFeeGwei + s.tipP10);
    data[3].push(s.baseFeeGwei + s.tipP50);
    data[4].push(s.baseFeeGwei);
  }
  // slide the window
  const cutoff = t - winSeconds[win.value];
  let drop = 0;
  while (drop < data[0].length && data[0][drop] < cutoff) drop++;
  if (drop > 0) data = data.map((col) => col.slice(drop));
  u.setData(data as uPlot.AlignedData);
}

const loading = ref(true);

async function load() {
  loading.value = true;
  try {
    const res = await api.history(win.value);
    data = toData(res.points);
    u?.setData(data as uPlot.AlignedData);
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  const width = el.value?.clientWidth || 600;
  u = new uPlot(makeOpts(width), [[], []] as unknown as uPlot.AlignedData, el.value!);
  await load();

  ro = new ResizeObserver((entries) => {
    const w = entries[0]?.contentRect.width;
    if (w && u) u.setSize({ width: w, height: 240 });
  });
  if (el.value) ro.observe(el.value);

  offStats = onStats(appendStats);
});

watch(win, load);

onBeforeUnmount(() => {
  offStats?.();
  ro?.disconnect();
  u?.destroy();
});
</script>

<template>
  <section class="section-rule">
    <header class="mb-1 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <h2 class="label">{{ title }}</h2>
        <span v-if="mode === 'size'" class="hidden items-center gap-3 font-mono text-[10px] muted sm:flex">
          <span class="flex items-center gap-1"><span class="h-0.5 w-3 bg-accent" /> pending</span>
          <span class="flex items-center gap-1"><span class="h-0.5 w-3 border-t border-dashed border-eth" /> queued</span>
        </span>
        <span v-else class="hidden items-center gap-3 font-mono text-[10px] muted sm:flex">
          <span class="flex items-center gap-1"><span class="h-2 w-3 bg-accent/15" /> p10–p90</span>
          <span class="flex items-center gap-1"><span class="h-0.5 w-3 bg-accent" /> median</span>
          <span class="flex items-center gap-1"><span class="h-0.5 w-3 border-t border-dashed border-eth" /> base</span>
        </span>
      </div>
      <div class="flex gap-1">
        <button
          v-for="w in (['1h', '6h', '24h'] as const)"
          :key="w"
          type="button"
          class="rounded px-2 py-0.5 font-mono text-[11px] transition-colors active:scale-[0.97]"
          :class="win === w ? 'bg-accent/15 text-accent' : 'text-neutral-500 hover:text-accent'"
          @click="win = w"
        >{{ w }}</button>
      </div>
    </header>
    <div ref="wrap" class="relative py-1">
      <div ref="el" />
      <div v-if="loading" class="absolute inset-0 my-1"><div class="skeleton h-full" /></div>
    </div>
  </section>
</template>

<style scoped>
:deep(.u-over) {
  cursor: crosshair;
}
</style>
