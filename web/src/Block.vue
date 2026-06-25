<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "./api";
import type { BlockMsg, TxLite } from "./types";
import { fmtGwei, fmtNum, fmtTime } from "./format";
import { builderName } from "./builders";
import HashLink from "./components/HashLink.vue";
import TxTable from "./components/TxTable.vue";

const route = useRoute();
const id = computed(() => String(route.params.id));

const block = ref<BlockMsg | null>(null);
const txs = ref<TxLite[]>([]);
const error = ref("");

async function load() {
  error.value = "";
  block.value = null;
  txs.value = [];
  try {
    const res = await api.block(id.value);
    block.value = res.block;
    txs.value = res.txs;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "failed to load";
  }
}

onMounted(load);
watch(id, load);

const utilPct = computed(() => (block.value ? Math.min(100, block.value.utilization * 100) : 0));
</script>

<template>
  <div class="space-y-6">
    <div v-if="error" class="border-t border-red-500/30 px-2 py-12 text-center">
      <div class="font-mono text-sm text-red-400">{{ error }}</div>
      <div class="mt-2 font-mono text-xs muted">block {{ id }}</div>
    </div>

    <template v-else-if="block">
      <header class="flex items-center gap-3">
        <h1 class="text-lg font-bold tracking-tight">
          Block <span class="font-mono tabular text-accent">#{{ fmtNum(block.number) }}</span>
        </h1>
        <div class="ml-auto flex gap-1.5">
          <router-link
            :to="`/block/${block.number - 1}`"
            class="rounded border border-accent/20 px-2.5 py-1 font-mono text-xs text-accent/70 transition-colors hover:border-accent/50 hover:text-accent active:scale-[0.97]"
          >← prev</router-link>
          <router-link
            :to="`/block/${block.number + 1}`"
            class="rounded border border-accent/20 px-2.5 py-1 font-mono text-xs text-accent/70 transition-colors hover:border-accent/50 hover:text-accent active:scale-[0.97]"
          >next →</router-link>
        </div>
      </header>

      <div class="break-all font-mono text-xs text-neutral-300">
        <HashLink :value="block.hash" :head="64" :tail="0" />
      </div>

      <!-- stats strip -->
      <section class="section-rule">
        <div class="grid grid-cols-2 gap-x-6 gap-y-3 text-xs sm:flex sm:flex-wrap sm:gap-0 sm:divide-x sm:divide-accent/15 sm:[&>div]:px-5 sm:[&>div:first-child]:pl-0">
          <div class="col-span-2 py-1 sm:col-span-1">
            <div class="label mb-1">Timestamp</div>
            <span class="break-words font-mono">{{ fmtTime(block.timestamp) }}</span>
          </div>
          <div class="py-1">
            <div class="label mb-1">Transactions</div>
            <span class="font-mono tabular">{{ block.txCount }}</span>
            <span class="muted ml-1.5 font-mono text-[10px]" title="seen in our mempool before inclusion">{{ block.knownCount }} seen pending</span>
          </div>
          <div class="py-1">
            <div class="label mb-1">Base fee</div>
            <span class="font-mono tabular">{{ fmtGwei(block.baseFeeGwei) }} <span class="muted">gwei</span></span>
          </div>
          <div class="col-span-2 py-1 sm:col-span-1">
            <div class="label mb-1">Builder</div>
            <router-link
              v-if="builderName(block.miner).known"
              :to="`/address/${block.miner}`"
              class="font-mono text-xs text-neutral-200 hover:text-accent"
              :title="block.miner"
            >{{ builderName(block.miner).name }}</router-link>
            <HashLink v-else :value="block.miner" :to="`/address/${block.miner}`" :head="6" :tail="4" class="text-xs" />
          </div>
          <div class="col-span-2 py-1 sm:min-w-56 sm:flex-1">
            <div class="label mb-1">Gas used</div>
            <div class="flex items-center gap-3">
              <span class="shrink-0 font-mono text-[11px] tabular">{{ fmtNum(block.gasUsed) }} / {{ fmtNum(block.gasLimit) }}</span>
              <div class="h-[3px] min-w-16 flex-1 overflow-hidden rounded-full bg-accent/10">
                <div
                  class="h-full rounded-full"
                  :class="utilPct > 90 ? 'bg-red-400' : utilPct > 60 ? 'bg-amber-400' : 'bg-accent'"
                  :style="{ width: `${utilPct}%` }"
                />
              </div>
              <span class="shrink-0 font-mono text-[11px] tabular muted">{{ utilPct.toFixed(1) }}%</span>
            </div>
          </div>
        </div>
      </section>

      <!-- txs -->
      <section class="section-rule">
        <h2 class="label mb-2">Transactions</h2>
        <TxTable v-if="txs.length" :txs="txs" />
        <div v-else class="px-2 py-8 text-center font-mono text-xs muted">
          no transactions recorded for this block
        </div>
      </section>
    </template>

    <div v-else class="space-y-4">
      <div class="skeleton h-8 w-56" />
      <div class="skeleton h-4 w-full max-w-xl" />
      <div class="skeleton h-16" />
      <div class="skeleton h-64" />
    </div>
  </div>
</template>
