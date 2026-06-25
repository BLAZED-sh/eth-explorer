<script setup lang="ts">
import { computed, ref } from "vue";
import { decodeCalldata, formatUnits, trimDecimals } from "../../decode";
import { fmtNum } from "../../format";
import HashLink from "../HashLink.vue";
import HexViewer from "./HexViewer.vue";

const props = defineProps<{ input: string; dataSize: number }>();

const decoded = computed(() => decodeCalldata(props.input));
// per-param toggle: raw uint vs ÷10^18 view
const scaled = ref<Record<string, boolean>>({});

function scaledValue(v: string): string {
  return trimDecimals(formatUnits(BigInt(v), 18), 6);
}
</script>

<template>
  <section class="border-t border-accent/15 pt-4">
    <header class="mb-3 flex items-center gap-2">
      <h2 class="label">Input data</h2>
      <span
        v-if="decoded"
        class="rounded border border-accent/25 bg-accent/10 px-2 py-0.5 font-mono text-[11px] font-semibold text-accent"
      >{{ decoded.name }}()</span>
      <span class="font-mono text-[10px] muted">{{ fmtNum(dataSize) }} bytes</span>
    </header>

    <template v-if="decoded">
      <div class="divide-y divide-accent/10 rounded-lg border border-accent/10 bg-bg/40">
        <div
          v-for="p in decoded.params"
          :key="p.name"
          class="grid grid-cols-[6rem_4rem_1fr] items-baseline gap-3 px-3 py-2 text-xs"
        >
          <span class="font-mono text-accent/80">{{ p.name }}</span>
          <span class="font-mono text-[10px] muted">{{ p.type }}</span>
          <span class="min-w-0">
            <HashLink v-if="p.type === 'address'" :value="p.value" :to="`/address/${p.value}`" :head="10" :tail="8" class="text-xs" />
            <span v-else class="flex flex-wrap items-baseline gap-2 font-mono tabular text-neutral-200">
              <span class="break-all">{{ scaled[p.name] ? scaledValue(p.value) : p.value }}</span>
              <button
                v-if="p.value.length > 6"
                type="button"
                class="shrink-0 rounded border border-accent/20 px-1.5 py-px font-mono text-[9px] text-accent/70 transition-colors hover:text-accent active:scale-[0.97]"
                title="token decimals are unknowable without a contract call — this assumes 18"
                @click="scaled[p.name] = !scaled[p.name]"
              >{{ scaled[p.name] ? "raw" : "÷10¹⁸" }}</button>
            </span>
          </span>
        </div>
        <div v-if="decoded.params.length === 0" class="px-3 py-2 font-mono text-[11px] muted">
          no arguments
        </div>
      </div>
      <details class="mt-2">
        <summary class="cursor-pointer font-mono text-[11px] text-accent/60 transition-colors hover:text-accent">raw calldata</summary>
        <HexViewer :hex="input" class="mt-2" />
      </details>
    </template>

    <HexViewer v-else :hex="input" />
  </section>
</template>
