<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "./api";
import type { ContractCode, TxLite } from "./types";
import { fmtBytes, fmtEth } from "./format";
import HashLink from "./components/HashLink.vue";
import TxTable from "./components/TxTable.vue";

const ContractCodeView = defineAsyncComponent(() => import("./components/contract/ContractCode.vue"));

const route = useRoute();
const address = computed(() => String(route.params.address).toLowerCase());

const txs = ref<TxLite[]>([]);
const nextCursor = ref<number | null>(null);
const loading = ref(true);

const code = ref<ContractCode | null>(null);

// Keep the first paint light; "load more" pulls larger pages.
const INITIAL_LIMIT = 10;
const PAGE_LIMIT = 50;

async function load(before?: number) {
  loading.value = true;
  try {
    const res = await api.address(address.value, before ? PAGE_LIMIT : INITIAL_LIMIT, before);
    txs.value = before ? [...txs.value, ...res.txs] : res.txs;
    nextCursor.value = res.nextCursor;
  } finally {
    loading.value = false;
  }
}

async function loadCode() {
  code.value = null;
  const addr = address.value;
  try {
    const res = await api.code(addr);
    // guard against a slow response landing after the route changed
    if (res.address === address.value) code.value = res;
  } catch {
    /* contract detection is best-effort — leave the page as a plain account */
  }
}

onMounted(() => {
  load();
  loadCode();
});
watch(address, () => {
  txs.value = [];
  load();
  loadCode();
});

const summary = computed(() => {
  let out = 0;
  let inn = 0;
  let volume = 0;
  for (const tx of txs.value) {
    if (tx.from === address.value) out++;
    else inn++;
    volume += tx.valueEth;
  }
  return { out, in: inn, volume };
});
</script>

<template>
  <div class="space-y-6">
    <header>
      <div class="flex items-center gap-2.5">
        <h1 class="text-lg font-bold tracking-tight">
          {{ code?.isContract ? "Contract" : "Address" }}
        </h1>
        <span
          v-if="code?.isContract"
          class="rounded border border-accent/40 bg-accent/10 px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-wider text-accent"
        >{{ fmtBytes(code.codeSize) }} code</span>
        <span
          v-else-if="code?.delegatedTo"
          class="rounded border border-purple-400/40 bg-purple-400/10 px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-wider text-purple-300"
        >EIP-7702</span>
        <span
          v-else-if="code"
          class="rounded border border-neutral-700 px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-wider text-neutral-500"
        >EOA</span>
      </div>
      <div class="mt-1 break-all font-mono text-sm text-neutral-200">
        <HashLink :value="address" :head="40" :tail="0" />
      </div>
      <div v-if="code?.delegatedTo" class="mt-1.5 font-mono text-[11px] muted">
        delegated EOA — execution runs the code at
        <HashLink :value="code.delegatedTo" :to="`/address/${code.delegatedTo}`" :head="6" :tail="4" class="text-purple-300/80" />
      </div>
      <div v-if="code?.creation" class="mt-1.5 font-mono text-[11px] muted">
        deployed by
        <HashLink :value="code.creation.deployer" :to="`/address/${code.creation.deployer}`" :head="6" :tail="4" class="text-accent/70" />
        in tx
        <HashLink :value="code.creation.txHash" :to="`/tx/${code.creation.txHash}`" :head="8" :tail="6" class="text-accent/70" />
        <template v-if="code.creation.blockNumber != null">
          @ block
          <router-link :to="`/block/${code.creation.blockNumber}`" class="text-accent/70 hover:text-accent">#{{ code.creation.blockNumber }}</router-link>
        </template>
      </div>
    </header>

    <!-- summary strip -->
    <section v-if="txs.length" class="section-rule">
      <div class="flex flex-wrap divide-x divide-accent/15 text-xs [&>div]:px-5 [&>div:first-child]:pl-0">
        <div class="py-1">
          <div class="label mb-1">Seen</div>
          <span class="font-mono tabular">{{ txs.length }}{{ nextCursor ? "+" : "" }} txs</span>
        </div>
        <div class="py-1">
          <div class="label mb-1">Out</div>
          <span class="font-mono tabular text-amber-400/90">{{ summary.out }}</span>
        </div>
        <div class="py-1">
          <div class="label mb-1">In</div>
          <span class="font-mono tabular text-green-400/90">{{ summary.in }}</span>
        </div>
        <div class="py-1">
          <div class="label mb-1">Volume</div>
          <span class="font-mono tabular">{{ fmtEth(summary.volume) }} <span class="muted">Ξ</span></span>
        </div>
        <div class="py-1">
          <div class="label mb-1">Window</div>
          <span class="font-mono text-[11px] muted">mempool retention only — not full chain history</span>
        </div>
      </div>
    </section>

    <section class="section-rule">
      <h2 class="label mb-2">Activity</h2>
      <TxTable v-if="txs.length" :txs="txs" show-status :highlight="address" />
      <div v-else-if="loading" class="py-2"><div class="skeleton h-64" /></div>
      <div v-else class="px-2 py-10 text-center font-mono text-xs muted">
        no recent mempool activity for this address
      </div>
    </section>

    <div v-if="nextCursor" class="text-center">
      <button
        type="button"
        :disabled="loading"
        class="rounded-lg border border-accent/30 bg-accent/5 px-5 py-2 font-mono text-xs text-accent transition-all hover:border-accent/60 hover:bg-accent/10 active:scale-[0.98] disabled:opacity-40"
        @click="load(nextCursor!)"
      >{{ loading ? "loading…" : "load more" }}</button>
    </div>

    <!-- contract functions + bytecode (lazy: pulls in sevm only here) -->
    <section v-if="code?.isContract" class="section-rule">
      <h2 class="label mb-2">Contract code</h2>
      <ContractCodeView :bytecode="code.bytecode" />
    </section>
  </div>
</template>
