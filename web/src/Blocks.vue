<script setup lang="ts">
import { onMounted, ref } from "vue";
import { api } from "./api";
import type { BlockMsg } from "./types";
import { fmtGwei, fmtNum } from "./format";
import { builderName } from "./builders";
import AgeTime from "./components/AgeTime.vue";

const blocks = ref<BlockMsg[]>([]);
const loading = ref(true);
const exhausted = ref(false);

async function load(before?: number) {
  loading.value = true;
  try {
    const res = await api.blocks(25, before);
    if (before) {
      blocks.value = [...blocks.value, ...res.blocks];
    } else {
      blocks.value = res.blocks;
    }
    if (res.blocks.length < 25) exhausted.value = true;
  } finally {
    loading.value = false;
  }
}

function loadMore() {
  const last = blocks.value[blocks.value.length - 1];
  if (last) load(last.number);
}

onMounted(() => load());

function knownPct(b: BlockMsg): string {
  if (b.txCount === 0) return "—";
  return `${Math.round((b.knownCount / b.txCount) * 100)}%`;
}
</script>

<template>
  <div class="stagger space-y-5">
    <h1 class="text-lg font-bold tracking-tight" style="--i: 0">Blocks</h1>

    <section class="section-rule overflow-x-auto" style="--i: 1">
      <table class="w-full min-w-[42rem] text-left text-xs">
        <thead>
          <tr class="thead-row border-b border-accent/15">
            <th class="px-4 py-2.5 font-semibold">Block</th>
            <th class="px-4 py-2.5 text-right font-semibold">Age</th>
            <th class="px-4 py-2.5 text-right font-semibold">Txs</th>
            <th class="px-4 py-2.5 text-right font-semibold" title="share of txs seen in our mempool before inclusion">Seen</th>
            <th class="px-4 py-2.5 font-semibold">Gas used</th>
            <th class="px-4 py-2.5 text-right font-semibold">Base fee</th>
            <th class="px-4 py-2.5 font-semibold">Builder</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="b in blocks" :key="b.number" class="trow">
            <td class="px-4 py-2.5">
              <router-link :to="`/block/${b.number}`" class="font-mono font-bold text-accent hover:underline">
                #{{ fmtNum(b.number) }}
              </router-link>
            </td>
            <td class="px-4 py-2.5 text-right font-mono muted">
              <AgeTime :unix-ms="b.timestamp * 1000" />
            </td>
            <td class="px-4 py-2.5 text-right font-mono tabular">{{ b.txCount }}</td>
            <td class="px-4 py-2.5 text-right font-mono tabular text-accent/80">{{ knownPct(b) }}</td>
            <td class="px-4 py-2.5">
              <div class="flex items-center gap-2">
                <div class="h-1 w-20 overflow-hidden rounded-full bg-accent/10">
                  <div
                    class="h-full rounded-full"
                    :class="b.utilization > 0.9 ? 'bg-red-400' : b.utilization > 0.6 ? 'bg-amber-400' : 'bg-accent'"
                    :style="{ width: `${Math.min(100, b.utilization * 100)}%` }"
                  />
                </div>
                <span class="font-mono tabular muted">{{ (b.utilization * 100).toFixed(0) }}%</span>
              </div>
            </td>
            <td class="px-4 py-2.5 text-right font-mono tabular">{{ fmtGwei(b.baseFeeGwei) }} <span class="muted text-[10px]">gw</span></td>
            <td class="px-4 py-2.5 font-mono">
              <router-link
                :to="`/address/${b.miner}`"
                class="hover:text-accent"
                :class="builderName(b.miner).known ? 'text-neutral-200' : 'text-neutral-400'"
                :title="b.miner"
              >{{ builderName(b.miner).name }}</router-link>
            </td>
          </tr>
          <tr v-if="loading && !blocks.length">
            <td colspan="7" class="p-4"><div class="skeleton h-64" /></td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="!exhausted && blocks.length" class="text-center" style="--i: 2">
      <button
        type="button"
        :disabled="loading"
        class="rounded-lg border border-accent/30 bg-accent/5 px-5 py-2 font-mono text-xs text-accent transition-all hover:border-accent/60 hover:bg-accent/10 active:scale-[0.98] disabled:opacity-40"
        @click="loadMore"
      >{{ loading ? "loading…" : "load more" }}</button>
    </div>
  </div>
</template>
