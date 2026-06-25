<script setup lang="ts">
import { computed } from "vue";
import type { RawReceipt, TxFull } from "../../types";
import { lifecycle } from "../../decode";
import { now } from "../../ticker";

const props = defineProps<{ tx: TxFull; receipt: RawReceipt | null }>();

const steps = computed(() => lifecycle(props.tx, props.receipt, now.value));

const dotCls: Record<string, string> = {
  done: "bg-accent border-accent",
  active: "bg-accent border-accent animate-pulse-dot",
  future: "bg-transparent border-accent/30",
  "terminal-ok": "bg-green-400 border-green-400",
  "terminal-bad": "bg-red-400 border-red-400",
  "terminal-warn": "bg-amber-400 border-amber-400",
};
const labelCls: Record<string, string> = {
  done: "text-neutral-200",
  active: "text-accent",
  future: "text-neutral-600",
  "terminal-ok": "text-green-400",
  "terminal-bad": "text-red-400",
  "terminal-warn": "text-amber-400",
};
</script>

<template>
  <div class="flex items-start">
    <div
      v-for="(step, i) in steps"
      :key="step.key"
      class="relative flex min-w-0 flex-1 flex-col items-center gap-1 px-1 text-center"
    >
      <!-- connector to the next dot: pinned at dot level and spanning
           center-to-center, so the labels below can't push it away -->
      <div
        v-if="i < steps.length - 1"
        class="absolute left-1/2 top-[5px] h-px w-full overflow-hidden bg-accent/15"
        aria-hidden="true"
      >
        <div class="absolute inset-0" :class="steps[i + 1].state === 'future' ? 'w-0' : 'bg-accent/60'" />
        <div
          v-if="step.state === 'active' || steps[i + 1].state === 'active'"
          class="absolute inset-0 connector-sweep"
        />
      </div>
      <!-- node (sits above the connector line) -->
      <span class="relative z-10 h-[11px] w-[11px] rounded-full border-2" :class="dotCls[step.state]" />
      <span class="relative z-10 text-xs font-semibold tracking-tight" :class="labelCls[step.state]">{{ step.label }}</span>
      <span class="relative z-10 font-mono text-[10px] muted">{{ step.detail }}</span>
    </div>
  </div>
</template>

<style scoped>
@keyframes sweep {
  from { transform: translateX(-100%); }
  to { transform: translateX(100%); }
}
.connector-sweep {
  background: linear-gradient(90deg, transparent, rgba(44, 209, 209, 0.9), transparent);
  animation: sweep 1.6s ease-in-out infinite;
}
@media (prefers-reduced-motion: reduce) {
  .connector-sweep { animation: none; }
}
</style>
