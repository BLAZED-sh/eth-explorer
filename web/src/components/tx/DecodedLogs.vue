<script setup lang="ts">
import { computed, ref } from "vue";
import type { RawLog } from "../../types";
import { decodeLogs, formatUnits, trimDecimals, type DecodedLog } from "../../decode";
import { shortAddr, shortHash } from "../../format";
import HashLink from "../HashLink.vue";

const props = defineProps<{ logs: RawLog[] }>();

const decoded = computed(() => decodeLogs(props.logs));
const known = computed(() => decoded.value.filter((l) => l.kind !== "unknown"));
const unknown = computed(() => decoded.value.filter((l) => l.kind === "unknown") as Extract<DecodedLog, { kind: "unknown" }>[]);
const showUnknown = ref(false);

const KIND_LABEL: Record<string, string> = {
  "erc20-transfer": "transfer",
  "erc721-transfer": "nft transfer",
  approval: "approval",
  "weth-deposit": "wrap",
  "weth-withdrawal": "unwrap",
};

function amount(v: bigint): string {
  return trimDecimals(formatUnits(v, 18), 6);
}
</script>

<template>
  <section class="border-t border-accent/15 pt-4">
    <header class="mb-3 flex items-center gap-2">
      <h2 class="label">Events</h2>
      <span class="font-mono text-[10px] muted">{{ logs.length }} log{{ logs.length === 1 ? "" : "s" }}</span>
    </header>

    <div v-if="known.length" class="divide-y divide-accent/10 rounded-lg border border-accent/10 bg-bg/40">
      <div v-for="(log, i) in known" :key="i" class="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2 text-xs">
        <span class="w-20 shrink-0 rounded border border-accent/20 bg-accent/5 px-1.5 py-0.5 text-center font-mono text-[10px] text-accent/80">
          {{ KIND_LABEL[log.kind] }}
        </span>

        <template v-if="log.kind === 'erc20-transfer' || log.kind === 'erc721-transfer'">
          <HashLink :value="log.from" :to="`/address/${log.from}`" :head="4" :tail="4" :copyable="false" class="text-xs" />
          <span class="text-accent/40">→</span>
          <HashLink :value="log.to" :to="`/address/${log.to}`" :head="4" :tail="4" :copyable="false" class="text-xs" />
          <span class="font-mono tabular text-neutral-200">
            <template v-if="log.kind === 'erc20-transfer'">{{ amount(log.value) }} <span class="muted" title="assuming 18 decimals">÷10¹⁸</span></template>
            <template v-else>#{{ log.tokenId }}</template>
          </span>
        </template>

        <template v-else-if="log.kind === 'approval'">
          <HashLink :value="log.owner" :to="`/address/${log.owner}`" :head="4" :tail="4" :copyable="false" class="text-xs" />
          <span class="text-accent/40">allows</span>
          <HashLink :value="log.spender" :to="`/address/${log.spender}`" :head="4" :tail="4" :copyable="false" class="text-xs" />
          <span class="font-mono tabular text-neutral-200">
            <template v-if="log.isNft">#{{ log.value }}</template>
            <template v-else-if="log.value > 2n ** 255n">∞</template>
            <template v-else>{{ amount(log.value) }} <span class="muted">÷10¹⁸</span></template>
          </span>
        </template>

        <template v-else>
          <HashLink :value="log.account" :to="`/address/${log.account}`" :head="4" :tail="4" :copyable="false" class="text-xs" />
          <span class="font-mono tabular text-neutral-200">{{ amount(log.value) }} <span class="muted">WETH</span></span>
        </template>

        <span class="ml-auto font-mono text-[10px] muted">
          via <router-link :to="`/address/${log.token}`" class="hover:text-accent">{{ shortAddr(log.token, 4, 4) }}</router-link>
        </span>
      </div>
    </div>

    <div v-if="unknown.length" class="mt-2">
      <button
        type="button"
        class="font-mono text-[11px] text-accent/60 transition-colors hover:text-accent active:scale-[0.99]"
        @click="showUnknown = !showUnknown"
      >{{ showUnknown ? "hide" : "show" }} {{ unknown.length }} other event{{ unknown.length === 1 ? "" : "s" }}</button>
      <div v-if="showUnknown" class="mt-2 divide-y divide-accent/10 rounded-lg border border-accent/10 bg-bg/40">
        <div v-for="(log, i) in unknown" :key="i" class="flex flex-wrap items-center gap-3 px-3 py-2 font-mono text-[11px]">
          <router-link :to="`/address/${log.address}`" class="text-neutral-300 hover:text-accent">{{ shortAddr(log.address) }}</router-link>
          <span class="muted break-all">{{ log.topic0 ? shortHash(log.topic0, 10, 6) : "anonymous" }}</span>
          <span class="muted ml-auto">{{ log.dataBytes }} B data</span>
        </div>
      </div>
    </div>

    <div v-if="!known.length && !unknown.length" class="font-mono text-xs muted">no events emitted</div>
  </section>
</template>
