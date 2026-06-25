<script setup lang="ts">
import type { TxLite } from "../types";
import { fmtEth, fmtGwei, methodLabel, shortAddr, shortHash } from "../format";
import AgeTime from "./AgeTime.vue";
import StatusBadge from "./StatusBadge.vue";

withDefaults(defineProps<{
  txs: TxLite[];
  showStatus?: boolean;
  /** lowercase address to mark in/out direction against */
  highlight?: string;
}>(), { showStatus: false, highlight: "" });
</script>

<template>
  <div class="overflow-x-auto">
    <table class="w-full min-w-[44rem] text-left text-xs">
      <thead>
        <tr class="thead-row border-b border-accent/15">
          <th class="px-3 py-2.5 font-semibold">Hash</th>
          <th v-if="showStatus" class="px-3 py-2.5 font-semibold">Status</th>
          <th class="px-3 py-2.5 font-semibold">Method</th>
          <th class="px-3 py-2.5 font-semibold">From</th>
          <th v-if="highlight" class="px-1 py-2.5" />
          <th class="px-3 py-2.5 font-semibold">To</th>
          <th class="px-3 py-2.5 text-right font-semibold">Value</th>
          <th class="px-3 py-2.5 text-right font-semibold">Gas</th>
          <th class="px-3 py-2.5 text-right font-semibold">Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="tx in txs" :key="tx.hash" class="trow">
          <td class="px-3 py-2.5">
            <router-link :to="`/tx/${tx.hash}`" class="font-mono text-accent hover:text-accent-hover hover:underline">
              {{ shortHash(tx.hash, 8, 6) }}
            </router-link>
          </td>
          <td v-if="showStatus" class="px-3 py-2.5">
            <StatusBadge :status="tx.status" />
          </td>
          <td class="px-3 py-2.5">
            <span
              class="inline-block rounded border px-1.5 py-0.5 font-mono text-[10px]"
              :class="tx.dataSize === 0
                ? 'border-eth/30 bg-eth/10 text-eth-muted'
                : 'border-accent/20 bg-accent/5 text-accent/80'"
            >{{ methodLabel(tx.methodSig, tx.dataSize) }}</span>
          </td>
          <td class="px-3 py-2.5 font-mono">
            <router-link
              v-if="tx.from !== highlight"
              :to="`/address/${tx.from}`"
              class="text-neutral-300 hover:text-accent"
            >{{ shortAddr(tx.from) }}</router-link>
            <span v-else class="text-neutral-400">{{ shortAddr(tx.from) }}</span>
          </td>
          <td v-if="highlight" class="px-1 py-2.5 text-center">
            <span
              class="inline-block rounded px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase"
              :class="tx.from === highlight
                ? 'bg-amber-500/15 text-amber-400'
                : 'bg-green-500/15 text-green-400'"
            >{{ tx.from === highlight ? "out" : "in" }}</span>
          </td>
          <td class="px-3 py-2.5 font-mono">
            <router-link
              v-if="tx.to && tx.to !== highlight"
              :to="`/address/${tx.to}`"
              class="text-neutral-300 hover:text-accent"
            >{{ shortAddr(tx.to) }}</router-link>
            <span v-else-if="tx.to" class="text-neutral-400">{{ shortAddr(tx.to) }}</span>
            <span v-else class="italic text-eth-muted">create</span>
          </td>
          <td class="px-3 py-2.5 text-right font-mono tabular text-neutral-200">
            <template v-if="tx.valueEth > 0">{{ fmtEth(tx.valueEth) }} <span class="muted">Ξ</span></template>
            <span v-else class="muted">—</span>
          </td>
          <td class="px-3 py-2.5 text-right font-mono tabular text-neutral-300">
            {{ fmtGwei(tx.gasPriceGwei) }} <span class="muted text-[10px]">gw</span>
          </td>
          <td class="px-3 py-2.5 text-right font-mono muted">
            <AgeTime :unix-ms="tx.firstSeen" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
