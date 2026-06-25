<script setup lang="ts">
import { computed } from "vue";
import type { TxLite } from "../types";
import { fmtEth, fmtGwei, methodLabel, shortAddr, shortHash } from "../format";
import AgeTime from "./AgeTime.vue";

const props = defineProps<{ tx: TxLite; mined?: boolean; confirmed?: boolean }>();

const method = computed(() => methodLabel(props.tx.methodSig, props.tx.dataSize));
const isPlainTransfer = computed(() => props.tx.dataSize === 0);
</script>

<template>
  <router-link
    :to="`/tx/${tx.hash}`"
    class="grid grid-cols-[7rem_1fr_4.5rem_3rem] items-center gap-3 px-2 py-2 text-[13px] transition-colors hover:bg-accent/5 sm:grid-cols-[7.5rem_5.5rem_1fr_5.5rem_5rem_3.5rem]"
    :class="mined && 'row-mined'"
  >
    <span class="truncate font-mono text-accent">{{ shortHash(tx.hash, 6, 4) }}</span>

    <span class="hidden truncate sm:block">
      <span
        class="inline-block max-w-full truncate rounded border px-1.5 py-px font-mono text-[10px]"
        :class="isPlainTransfer
          ? 'border-eth/30 bg-eth/10 text-eth-muted'
          : 'border-accent/20 bg-accent/5 text-accent/80'"
      >{{ method }}</span>
    </span>

    <span class="truncate font-mono text-neutral-300">
      {{ shortAddr(tx.from, 4, 4) }}
      <span class="mx-1" :class="confirmed ? 'text-green-400/70' : 'text-accent/40'">→</span>
      <span v-if="tx.to">{{ shortAddr(tx.to, 4, 4) }}</span>
      <span v-else class="italic text-eth-muted">create</span>
    </span>

    <span class="hidden text-right font-mono tabular text-neutral-200 sm:block">
      <template v-if="tx.valueEth > 0">{{ fmtEth(tx.valueEth) }} <span class="muted">Ξ</span></template>
      <span v-else class="muted">—</span>
    </span>

    <span class="text-right font-mono tabular text-neutral-300">
      {{ fmtGwei(tx.gasPriceGwei) }} <span class="muted text-[10px]">gw</span>
    </span>

    <span class="text-right font-mono text-[11px] muted">
      <AgeTime :unix-ms="tx.firstSeen" />
    </span>
  </router-link>
</template>
