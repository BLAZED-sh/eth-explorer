<script setup lang="ts">
import { computed, ref } from "vue";
import { store, setPaused } from "../store";
import TxRow from "./TxRow.vue";

type Tab = "pending" | "confirmed";
const tab = ref<Tab>("pending");

const rows = computed(() => (tab.value === "pending" ? store.feed : store.confirmed));

function enter() {
  if (tab.value === "pending") setPaused(true);
}
function leave() {
  if (tab.value === "pending") setPaused(false);
}
function select(t: Tab) {
  if (t === tab.value) return;
  // leaving pending must not strand the feed in a paused state
  if (tab.value === "pending") setPaused(false);
  tab.value = t;
}
</script>

<template>
  <section class="section-rule flex flex-col">
    <header class="mb-2 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <span class="h-2 w-2 shrink-0 rounded-full" :class="store.connected ? 'animate-pulse-dot bg-accent' : 'bg-red-500'" />
        <div class="relative inline-flex items-center rounded-lg border border-accent/15 bg-bg/50 p-0.5">
          <!-- sliding highlight: translates + morphs colour between the two tabs -->
          <span
            aria-hidden="true"
            class="pointer-events-none absolute inset-y-0.5 left-0.5 w-[5.5rem] rounded-md transition-all duration-300 ease-[cubic-bezier(0.22,1,0.36,1)]"
            :class="tab === 'pending' ? 'translate-x-0 bg-accent/15' : 'translate-x-full bg-green-400/15'"
          />
          <button
            type="button"
            class="relative z-10 w-[5.5rem] rounded-md py-1 text-center font-mono text-[11px] font-semibold uppercase tracking-wider transition-colors active:scale-[0.98]"
            :class="tab === 'pending' ? 'text-accent' : 'text-neutral-500 hover:text-neutral-300'"
            @click="select('pending')"
          >Pending</button>
          <button
            type="button"
            class="relative z-10 w-[5.5rem] rounded-md py-1 text-center font-mono text-[11px] font-semibold uppercase tracking-wider transition-colors active:scale-[0.98]"
            :class="tab === 'confirmed' ? 'text-green-400' : 'text-neutral-500 hover:text-neutral-300'"
            @click="select('confirmed')"
          >Confirmed</button>
        </div>
        <span v-if="tab === 'pending' && store.stats" class="font-mono text-[11px] muted tabular">
          {{ store.stats.txPerSec.toFixed(1) }} tx/s
        </span>
      </div>
      <div class="flex items-center gap-3">
        <transition name="fade">
          <span v-if="tab === 'pending' && store.paused" class="rounded border border-amber-500/40 bg-amber-500/10 px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider text-amber-400">
            paused<template v-if="store.pausedBuffer.length"> · {{ store.pausedBuffer.length }} new</template>
          </span>
        </transition>
        <a
          href="https://mempool.blazed.sh"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-1.5 rounded-md border border-accent/40 bg-accent/10 px-3 py-1.5 font-mono text-xs font-semibold uppercase tracking-wider text-accent transition-colors hover:border-accent/70 hover:bg-accent/20 active:scale-[0.98]"
          title="Detailed mempool insights at mempool.blazed.sh"
        >
          <span class="sm:hidden">mempool</span>
          <span class="hidden sm:inline">full mempool insights</span>
          <span class="hidden text-sm leading-none sm:inline">↗</span>
        </a>
      </div>
    </header>

    <div class="grid grid-cols-[7rem_1fr_4.5rem_3rem] gap-3 border-b border-accent/15 px-2 py-1.5 font-mono text-[10px] font-semibold uppercase tracking-wider text-accent/60 sm:grid-cols-[7.5rem_5.5rem_1fr_5.5rem_5rem_3.5rem]">
      <span>Hash</span>
      <span class="hidden sm:block">Method</span>
      <span>From → To</span>
      <span class="hidden text-right sm:block">Value</span>
      <span class="text-right">Gas</span>
      <span class="text-right">Age</span>
    </div>

    <!-- fixed-height window: row churn inside here never reflows the page
         (prevents the scroll position from jumping as txs stream in) -->
    <div
      class="relative h-[23rem] overflow-hidden"
      @mouseenter="enter"
      @mouseleave="leave"
    >
      <TransitionGroup v-if="rows.length" :key="tab" name="feed" tag="div" class="relative divide-y divide-accent/[0.06]">
        <TxRow
          v-for="tx in rows"
          :key="tx.hash"
          :tx="tx"
          :confirmed="tab === 'confirmed'"
          :mined="tab === 'pending' && store.justMined.has(tx.hash)"
          :class="tab === 'pending' ? 'row-flash' : 'row-confirmed'"
        />
      </TransitionGroup>

      <div v-else class="flex flex-col gap-1.5 py-2">
        <div v-for="i in 10" :key="i" class="skeleton h-6" />
      </div>
    </div>
  </section>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
