<script setup lang="ts">
import { computed } from "vue";
import type { BlockMsg } from "../types";
import { fmtGwei, fmtNum } from "../format";
import { builderName } from "../builders";
import AgeTime from "./AgeTime.vue";

const props = defineProps<{ block: BlockMsg }>();

const builder = computed(() => builderName(props.block.miner));
const utilPct = computed(() => Math.min(100, props.block.utilization * 100));
const utilColor = computed(() => {
  if (utilPct.value > 90) return "bg-red-400";
  if (utilPct.value > 60) return "bg-amber-400";
  return "bg-accent";
});
</script>

<template>
  <router-link
    :to="`/block/${block.number}`"
    class="group block px-1 py-2.5 transition-colors hover:bg-accent/5 active:scale-[0.99]"
  >
    <div class="flex items-baseline justify-between gap-2">
      <span class="flex min-w-0 items-baseline gap-2">
        <span class="font-mono text-sm font-bold tabular text-accent group-hover:text-accent-hover">
          #{{ fmtNum(block.number) }}
        </span>
        <span
          class="truncate font-mono text-[11px]"
          :class="builder.known ? 'text-neutral-300' : 'muted'"
          :title="block.miner"
        >{{ builder.name }}</span>
      </span>
      <span class="shrink-0 font-mono text-[10px] muted"><AgeTime :unix-ms="block.timestamp * 1000" /> ago</span>
    </div>
    <div class="mt-1.5 flex items-center gap-3">
      <div class="h-[3px] flex-1 overflow-hidden rounded-full bg-accent/10" :title="`${utilPct.toFixed(0)}% gas used`">
        <div class="h-full rounded-full transition-all duration-500" :class="utilColor" :style="{ width: `${utilPct}%` }" />
      </div>
      <span class="shrink-0 font-mono text-[10px] tabular text-neutral-400">{{ block.txCount }} txs</span>
      <span class="shrink-0 font-mono text-[10px] tabular muted">{{ fmtGwei(block.baseFeeGwei) }} gw</span>
    </div>
  </router-link>
</template>
