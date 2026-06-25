<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, ref, watch } from "vue";
import { store } from "../store";
import { fmtGwei, fmtNum } from "../format";

const MempoolStream = defineAsyncComponent(() => import("./MempoolStream.vue"));

// Eased display value for the big readout.
const shownGas = ref(0);
let raf = 0;
watch(
  () => store.gas?.standard ?? 0,
  (target) => {
    cancelAnimationFrame(raf);
    const from = shownGas.value;
    const start = performance.now();
    const dur = 600;
    const step = (t: number) => {
      const k = Math.min(1, (t - start) / dur);
      const e = 1 - Math.pow(1 - k, 3);
      shownGas.value = from + (target - from) * e;
      if (k < 1) raf = requestAnimationFrame(step);
    };
    raf = requestAnimationFrame(step);
  },
  { immediate: true },
);
onBeforeUnmount(() => cancelAnimationFrame(raf));

const tiers = computed(() => {
  const g = store.gas;
  if (!g) return null;
  return [
    { label: "slow", value: g.slow },
    { label: "standard", value: g.standard },
    { label: "fast", value: g.fast },
    { label: "rapid", value: g.rapid },
  ];
});

// Counters: live stats tick when available, hello gas snapshot until then.
const pending = computed(() => store.stats?.pending ?? store.gas?.pendingCount ?? null);
const queued = computed(() => store.stats?.queued ?? store.gas?.queuedCount ?? 0);
const txPerSec = computed(() => store.stats?.txPerSec ?? null);
// Only claim "probably more" once we've actually polled the node's txpool and
// found it unavailable — i.e. the count is our tracked-pool floor, not exact.
const pendingCapped = computed(() => store.stats != null && store.stats.pendingExact === false);
</script>

<template>
  <section class="relative h-56 overflow-hidden rounded-2xl border border-accent/15 lg:h-72">
    <!-- layer 0: static fallback, zero JS -->
    <div class="absolute inset-0 hero-fallback" aria-hidden="true" />
    <div class="bg-grid absolute inset-0" aria-hidden="true" />

    <!-- layer 1: calm live mempool stream -->
    <MempoolStream />

    <!-- layer 2: readout overlay, left-aligned -->
    <div class="pointer-events-none absolute inset-0 flex flex-col justify-between p-5 sm:p-7">
      <div>
        <div class="label">standard gas · gwei</div>
        <div class="mt-1 flex items-baseline gap-3">
          <span class="font-mono text-6xl font-bold tabular leading-none tracking-tighter text-neutral-50 lg:text-8xl">
            {{ store.gas ? fmtGwei(shownGas) : "—" }}
          </span>
        </div>
        <div v-if="store.gas" class="mt-2 font-mono text-xs muted">
          base {{ fmtGwei(store.gas.baseFeeGwei) }} gwei
        </div>
      </div>

      <div class="flex items-end justify-between gap-4">
        <!-- tier strip -->
        <div v-if="tiers" class="flex divide-x divide-accent/20 rounded-lg border border-accent/15 bg-bg/50 backdrop-blur-sm">
          <div v-for="t in tiers" :key="t.label" class="px-3 py-2 sm:px-4">
            <div class="font-mono text-[9px] uppercase tracking-widest muted">{{ t.label }}</div>
            <div class="mt-0.5 font-mono text-sm font-bold tabular text-neutral-100 sm:text-base"
              :class="t.label === 'rapid' && 'text-accent'"
            >{{ fmtGwei(t.value) }}</div>
          </div>
        </div>
        <div v-else class="skeleton h-14 w-72" />

        <!-- live counters, bottom right -->
        <div class="hidden flex-col items-end gap-1 font-mono text-[11px] text-neutral-400 sm:flex">
          <span v-if="pending != null" class="tabular">
            <span class="text-neutral-100">{{ fmtNum(pending) }}<span v-if="pendingCapped" class="text-amber-400">+</span></span> pending<template v-if="queued > 0"> ·
            <span class="text-neutral-100">{{ fmtNum(queued) }}</span> queued</template>
          </span>
          <span
            v-if="pendingCapped"
            class="text-[10px] text-amber-400/80"
            title="The node's txpool count is unavailable, so this is only the txs we've tracked — the real mempool is larger."
          >probably more</span>
          <span v-if="txPerSec != null" class="tabular">{{ txPerSec.toFixed(1) }} tx/s</span>
          <span v-if="store.head" class="tabular text-accent/80">block {{ fmtNum(store.head) }}</span>
        </div>
      </div>
    </div>

    <!-- legend -->
    <div class="pointer-events-none absolute right-3 top-3 hidden font-mono text-[9px] uppercase tracking-widest text-accent/40 sm:block">
      each dot = a pending tx flowing toward the next block · brighter = higher gas tip
    </div>
  </section>
</template>

<style scoped>
.hero-fallback {
  background:
    radial-gradient(ellipse 70% 90% at 70% 50%, rgba(44, 209, 209, 0.07) 0%, transparent 60%),
    radial-gradient(ellipse 50% 70% at 25% 80%, rgba(98, 126, 234, 0.05) 0%, transparent 65%),
    linear-gradient(180deg, rgba(0, 18, 18, 0.6), rgba(0, 8, 8, 0.9));
}
</style>
