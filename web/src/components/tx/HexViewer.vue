<script setup lang="ts">
import { computed, ref } from "vue";
import { toWords } from "../../decode";

const props = withDefaults(defineProps<{
  hex: string;
  collapsedRows?: number;
}>(), { collapsedRows: 8 });

const parsed = computed(() => toWords(props.hex));
const expanded = ref(false);
const visible = computed(() =>
  expanded.value ? parsed.value.words : parsed.value.words.slice(0, props.collapsedRows),
);
const hiddenCount = computed(() => parsed.value.words.length - visible.value.length);

function fmtOffset(n: number): string {
  return "0x" + n.toString(16).padStart(3, "0");
}
</script>

<template>
  <div class="overflow-x-auto rounded-lg border border-accent/10 bg-bg/70">
    <div v-if="parsed.selector" class="flex items-center gap-3 border-b border-accent/15 px-3 py-1.5">
      <span class="w-12 shrink-0 font-mono text-[10px] muted">sig</span>
      <code class="font-mono text-[11px] font-bold text-accent">{{ parsed.selector }}</code>
    </div>
    <div
      v-for="(w, i) in visible"
      :key="w.offset"
      class="flex items-center gap-3 px-3 py-1"
      :class="i % 2 === 1 && 'bg-accent/[0.03]'"
    >
      <span class="w-12 shrink-0 select-none font-mono text-[10px] muted">{{ fmtOffset(w.offset) }}</span>
      <code class="break-all font-mono text-[11px] leading-relaxed text-neutral-400">{{ w.hex }}</code>
    </div>
    <button
      v-if="hiddenCount > 0 || expanded"
      type="button"
      class="w-full border-t border-accent/10 px-3 py-1.5 text-left font-mono text-[11px] text-accent/70 transition-colors hover:text-accent active:scale-[0.99]"
      @click="expanded = !expanded"
    >{{ expanded ? "collapse" : `show ${hiddenCount} more words` }}</button>
  </div>
</template>
