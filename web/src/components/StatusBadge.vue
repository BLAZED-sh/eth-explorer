<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{ status: string }>();

const cls = computed(() => {
  switch (props.status) {
    case "pending":
      return "border-accent/40 bg-accent/10 text-accent";
    case "mined":
      return "border-green-500/40 bg-green-500/10 text-green-400";
    case "dropped":
      return "border-red-500/40 bg-red-500/10 text-red-400";
    case "replaced":
      return "border-amber-500/40 bg-amber-500/10 text-amber-400";
    default:
      return "border-neutral-500/40 bg-neutral-500/10 text-neutral-400";
  }
});
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 font-mono text-[11px] font-semibold uppercase tracking-wider"
    :class="cls"
  >
    <span v-if="status === 'pending'" class="h-1.5 w-1.5 animate-pulse rounded-full bg-accent" />
    {{ status }}
  </span>
</template>
