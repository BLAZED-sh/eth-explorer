<script setup lang="ts">
import { store } from "./store";
import MempoolHero from "./components/MempoolHero.vue";
import LiveTxFeed from "./components/LiveTxFeed.vue";
import BlockRow from "./components/BlockRow.vue";
import MempoolChart from "./components/MempoolChart.vue";
</script>

<template>
  <div class="stagger space-y-6">
    <MempoolHero style="--i: 0" />

    <!-- feed + blocks rail: asymmetric 2fr/1fr -->
    <div class="grid gap-x-8 gap-y-6 lg:grid-cols-[2fr_1fr]" style="--i: 1">
      <LiveTxFeed />

      <aside class="section-rule">
        <div class="mb-2 flex items-center justify-between">
          <h2 class="label">Blocks</h2>
          <router-link
            to="/blocks"
            class="font-mono text-[11px] text-accent/70 transition-colors hover:text-accent active:scale-[0.98]"
          >all →</router-link>
        </div>
        <TransitionGroup v-if="store.blocks.length" name="feed" tag="div" class="relative divide-y divide-accent/[0.08]">
          <BlockRow v-for="b in store.blocks.slice(0, 10)" :key="b.number" :block="b" />
        </TransitionGroup>
        <div v-else class="space-y-1.5 py-2">
          <div v-for="i in 8" :key="i" class="skeleton h-10" />
        </div>
      </aside>
    </div>

    <!-- charts -->
    <div class="grid gap-x-8 gap-y-6 xl:grid-cols-2" style="--i: 2">
      <MempoolChart mode="size" title="Mempool size" />
      <MempoolChart mode="fees" title="Gas price" />
    </div>
  </div>
</template>
