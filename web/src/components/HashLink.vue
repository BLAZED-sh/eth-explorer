<script setup lang="ts">
import { computed, ref } from "vue";
import { shortHash } from "../format";

const props = withDefaults(defineProps<{
  value: string;
  to?: string;          // router target; omit for plain text
  head?: number;
  tail?: number;
  copyable?: boolean;
}>(), { head: 8, tail: 6, copyable: true });

const display = computed(() => shortHash(props.value, props.head, props.tail));
const copied = ref(false);

async function copy() {
  try {
    await navigator.clipboard.writeText(props.value);
    copied.value = true;
    setTimeout(() => (copied.value = false), 1200);
  } catch { /* clipboard unavailable */ }
}
</script>

<template>
  <span class="group/hash inline-flex items-center gap-1 font-mono">
    <router-link
      v-if="to"
      :to="to"
      class="text-accent transition-colors hover:text-accent-hover hover:underline decoration-accent/40 underline-offset-2"
      :title="value"
    >{{ display }}</router-link>
    <span v-else class="text-neutral-300" :title="value">{{ display }}</span>
    <button
      v-if="copyable"
      type="button"
      class="opacity-0 transition-opacity group-hover/hash:opacity-100"
      :class="copied ? '!opacity-100 text-green-400' : 'text-accent/50 hover:text-accent'"
      :title="copied ? 'copied' : 'copy'"
      @click.stop.prevent="copy"
    >
      <svg v-if="!copied" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
        <rect x="9" y="9" width="11" height="11" rx="2" />
        <path d="M5 15V5a2 2 0 0 1 2-2h10" />
      </svg>
      <svg v-else class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
        <path d="m4 12 5 5L20 6" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </button>
  </span>
</template>
