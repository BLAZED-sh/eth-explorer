<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";

// Matrix-style scramble that decodes to "404".
const text = ref("");
const GLYPHS = "0123456789abcdef░▒▓";
const TARGET = "404";
let timer = 0;

onMounted(() => {
  let frame = 0;
  timer = window.setInterval(() => {
    frame++;
    text.value = TARGET.split("")
      .map((ch, i) => (frame > i * 6 + 10 ? ch : GLYPHS[Math.floor(Math.random() * GLYPHS.length)]))
      .join("");
    if (frame > TARGET.length * 6 + 10) clearInterval(timer);
  }, 50);
});
onBeforeUnmount(() => clearInterval(timer));
</script>

<template>
  <div class="flex flex-col items-start justify-center border-t border-accent/15 py-24 pl-[8vw]">
    <div class="font-mono text-7xl font-bold tabular text-accent sm:text-8xl">{{ text || "▒▒▒" }}</div>
    <p class="mt-4 font-mono text-sm muted">nothing at this path — it may have been dropped from the pool</p>
    <router-link
      to="/"
      class="mt-8 rounded-lg border border-accent/30 bg-accent/5 px-5 py-2 font-mono text-xs text-accent transition-all hover:border-accent/60 hover:bg-accent/10 active:scale-[0.98]"
    >← back to the mempool</router-link>
  </div>
</template>
