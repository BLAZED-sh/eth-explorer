<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { api } from "../api";

const router = useRouter();
const q = ref("");
const notFound = ref(false);
const searching = ref(false);
const inputEl = ref<HTMLInputElement | null>(null);

function onKeydown(e: KeyboardEvent) {
  if (e.key !== "/" || e.metaKey || e.ctrlKey || e.altKey) return;
  const target = e.target as HTMLElement | null;
  if (target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable)) return;
  e.preventDefault();
  inputEl.value?.focus();
}
onMounted(() => document.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => document.removeEventListener("keydown", onKeydown));

async function go() {
  const query = q.value.trim();
  if (!query || searching.value) return;
  searching.value = true;
  notFound.value = false;
  try {
    const res = await api.search(query);
    switch (res.type) {
      case "tx":
        router.push(`/tx/${res.ref}`);
        break;
      case "address":
        router.push(`/address/${res.ref}`);
        break;
      case "block":
        router.push(`/block/${res.ref}`);
        break;
      default:
        notFound.value = true;
        setTimeout(() => (notFound.value = false), 2000);
        return;
    }
    q.value = "";
  } catch {
    notFound.value = true;
    setTimeout(() => (notFound.value = false), 2000);
  } finally {
    searching.value = false;
  }
}
</script>

<template>
  <form class="relative min-w-0 flex-1 sm:max-w-xs" @submit.prevent="go">
    <svg
      class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2"
      :class="notFound ? 'text-red-400' : 'text-accent/50'"
      viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true"
    >
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" stroke-linecap="round" />
    </svg>
    <input
      ref="inputEl"
      v-model="q"
      type="search"
      spellcheck="false"
      autocomplete="off"
      :placeholder="notFound ? 'no result' : 'tx / address / block'"
      class="w-full rounded-md border bg-bg/60 py-1.5 pl-8 pr-8 font-mono text-xs text-neutral-200 outline-none transition-colors placeholder:text-neutral-600"
      :class="notFound
        ? 'border-red-500/60 placeholder:text-red-400/70'
        : 'border-accent/20 focus:border-accent/60'"
    />
    <kbd
      class="pointer-events-none absolute right-2 top-1/2 hidden -translate-y-1/2 rounded border border-accent/20 bg-accent/5 px-1.5 font-mono text-[10px] leading-4 text-accent/50 sm:block"
    >/</kbd>
  </form>
</template>
