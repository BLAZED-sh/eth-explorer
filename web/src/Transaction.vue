<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "./api";
import type { TxDetail } from "./types";
import { fmtDuration, fmtEth, fmtNum, txTypeLabel } from "./format";
import { formatEthExact, gweiToWei, trimDecimals } from "./decode";
import HashLink from "./components/HashLink.vue";
import StatusBadge from "./components/StatusBadge.vue";
import TxLifecycle from "./components/tx/TxLifecycle.vue";
import FeeBreakdown from "./components/tx/FeeBreakdown.vue";
import DecodedCall from "./components/tx/DecodedCall.vue";
import DecodedLogs from "./components/tx/DecodedLogs.vue";

const route = useRoute();
const hash = computed(() => String(route.params.hash));

const detail = ref<TxDetail | null>(null);
const error = ref("");
let pollTimer: number | undefined;

async function load() {
  error.value = "";
  try {
    detail.value = await api.tx(hash.value);
    // Keep polling while it's still pending so the page flips live.
    clearTimeout(pollTimer);
    if (detail.value.tx.status === "pending") {
      pollTimer = window.setTimeout(load, 6000);
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : "failed to load";
  }
}

onMounted(load);
watch(hash, () => {
  detail.value = null;
  load();
});
onBeforeUnmount(() => clearTimeout(pollTimer));

const tx = computed(() => detail.value?.tx ?? null);

// Pending estimate: max cost ≤ gasLimit × maxFee (or gasPrice for legacy).
const maxCostEth = computed(() => {
  if (!tx.value) return null;
  const priceGwei = tx.value.type >= 2 ? tx.value.maxFeeGwei : tx.value.gasPriceGwei;
  if (priceGwei <= 0) return null;
  return trimDecimals(formatEthExact(BigInt(tx.value.gasLimit) * gweiToWei(priceGwei)), 8);
});
</script>

<template>
  <div class="mx-auto max-w-4xl">
    <div v-if="error" class="border-t border-red-500/30 px-2 py-12 text-center">
      <div class="font-mono text-sm text-red-400">{{ error }}</div>
      <div class="mt-2 break-all font-mono text-xs muted">{{ hash }}</div>
    </div>

    <template v-else-if="tx && detail">
      <header class="flex flex-wrap items-center gap-3">
        <h1 class="text-lg font-bold tracking-tight">Transaction</h1>
        <StatusBadge :status="tx.status" />
        <span
          v-if="tx.confirmMs != null"
          class="rounded-full border border-green-500/30 bg-green-500/5 px-2.5 py-0.5 font-mono text-[11px] text-green-400"
          title="time from first seen in mempool to inclusion"
        >confirmed in {{ fmtDuration(tx.confirmMs) }}</span>
        <span class="ml-auto hidden font-mono text-[11px] muted sm:block">{{ txTypeLabel(tx.type) }}</span>
      </header>

      <div class="mt-2 break-all font-mono text-xs text-neutral-300">
        <HashLink :value="tx.hash" :head="64" :tail="0" />
      </div>

      <div class="mt-6 space-y-6">
        <!-- lifecycle -->
        <section class="border-t border-accent/15 pt-4">
          <h2 class="label mb-4">Lifecycle</h2>
          <TxLifecycle :tx="tx" :receipt="detail.receipt" />
        </section>

        <!-- core details -->
        <section class="border-t border-accent/15 pt-4">
          <h2 class="label mb-2">Details</h2>
          <dl class="divide-y divide-accent/10 text-xs">
            <div class="grid grid-cols-[7rem_1fr] items-baseline gap-3 py-2 sm:grid-cols-[9rem_1fr]">
              <dt class="muted">From</dt>
              <dd><HashLink :value="tx.from" :to="`/address/${tx.from}`" :head="12" :tail="10" class="text-xs" /></dd>
            </div>
            <div class="grid grid-cols-[7rem_1fr] items-baseline gap-3 py-2 sm:grid-cols-[9rem_1fr]">
              <dt class="muted">To</dt>
              <dd>
                <HashLink v-if="tx.to" :value="tx.to" :to="`/address/${tx.to}`" :head="12" :tail="10" class="text-xs" />
                <span v-else class="font-mono italic text-eth-muted">contract creation</span>
              </dd>
            </div>
            <div class="grid grid-cols-[7rem_1fr] items-baseline gap-3 py-2 sm:grid-cols-[9rem_1fr]">
              <dt class="muted">Value</dt>
              <dd class="font-mono tabular text-sm font-bold text-neutral-100">
                {{ fmtEth(tx.valueEth) }} <span class="font-normal text-eth-muted">ETH</span>
              </dd>
            </div>
            <div class="grid grid-cols-[7rem_1fr] items-baseline gap-3 py-2 sm:grid-cols-[9rem_1fr]">
              <dt class="muted">Nonce</dt>
              <dd class="font-mono tabular">{{ tx.nonce }}</dd>
            </div>
            <div class="grid grid-cols-[7rem_1fr] items-baseline gap-3 py-2 sm:grid-cols-[9rem_1fr]">
              <dt class="muted">First seen</dt>
              <dd class="font-mono">
                {{ new Date(tx.firstSeen).toLocaleString() }}
                <span v-if="!tx.seenInMempool" class="muted ml-1 text-[10px]">(observed in block, not mempool)</span>
              </dd>
            </div>
            <div class="grid grid-cols-[7rem_1fr] items-baseline gap-3 py-2 sm:grid-cols-[9rem_1fr]">
              <dt class="muted">Block</dt>
              <dd class="font-mono">
                <template v-if="tx.blockNumber != null">
                  <router-link :to="`/block/${tx.blockNumber}`" class="text-accent hover:underline">
                    #{{ fmtNum(tx.blockNumber) }}
                  </router-link>
                  <span v-if="tx.blockPosition != null" class="muted ml-2 text-[10px]">position {{ tx.blockPosition }}</span>
                </template>
                <span v-else class="muted">—</span>
              </dd>
            </div>
            <div v-if="tx.replacedBy" class="grid grid-cols-[7rem_1fr] items-baseline gap-3 py-2 sm:grid-cols-[9rem_1fr]">
              <dt class="muted">Replaced by</dt>
              <dd><HashLink :value="tx.replacedBy" :to="`/tx/${tx.replacedBy}`" :head="12" :tail="10" class="text-xs" /></dd>
            </div>
          </dl>
        </section>

        <!-- fees -->
        <FeeBreakdown v-if="detail.receipt" :tx="tx" :receipt="detail.receipt" />
        <section v-else-if="tx.status === 'pending'" class="border-t border-accent/15 pt-4">
          <h2 class="label mb-2">Fee estimate</h2>
          <p class="font-mono text-xs text-neutral-300">
            max cost ≤ <span class="tabular font-bold">{{ maxCostEth ?? "?" }}</span> <span class="muted">ETH</span>
            <span class="muted ml-2">({{ fmtNum(tx.gasLimit) }} gas × {{ tx.type >= 2 ? tx.maxFeeGwei.toFixed(2) + " max fee" : tx.gasPriceGwei.toFixed(2) + " gas price" }} gwei)</span>
          </p>
        </section>

        <!-- calldata -->
        <DecodedCall v-if="tx.dataSize > 0 && detail.input && detail.input !== '0x'" :input="detail.input" :data-size="tx.dataSize" />
        <section v-else-if="tx.dataSize > 0" class="border-t border-accent/15 pt-4">
          <h2 class="label mb-2">Input data</h2>
          <p class="font-mono text-xs muted">calldata unavailable — tx evicted from the node ({{ fmtNum(tx.dataSize) }} bytes, selector {{ tx.methodSig || "unknown" }})</p>
        </section>

        <!-- events -->
        <DecodedLogs v-if="detail.receipt?.logs?.length" :logs="detail.receipt.logs" />
      </div>
    </template>

    <div v-else class="space-y-4">
      <div class="skeleton h-7 w-56" />
      <div class="skeleton h-4 w-full max-w-xl" />
      <div class="skeleton h-20" />
      <div class="skeleton h-64" />
    </div>
  </div>
</template>
