<script setup lang="ts">
import { store } from "../store";
import { fmtNum } from "../format";
import SearchBox from "./SearchBox.vue";
</script>

<template>
  <header class="glass-effect sticky top-0 z-50">
    <div class="mx-auto flex h-14 w-full max-w-7xl items-center gap-4 px-4 sm:px-6">
      <router-link to="/" class="flex shrink-0 items-center gap-2 text-[15px] font-bold tracking-tight">
        <img
          src="/blazed-mark.png"
          alt="BLAZED.sh"
          width="24"
          height="24"
          class="h-6 w-6 shrink-0 drop-shadow-[0_0_6px_rgba(44,209,209,0.35)]"
        />
        <span class="text-gradient">blazed.sh</span>
        <span class="flex items-center gap-1">
          <img src="/eth.svg" alt="Ethereum" width="14" height="14" class="h-3.5 w-3.5 shrink-0 opacity-80" />
          <span class="font-normal text-neutral-300">explorer</span>
        </span>
      </router-link>

      <nav class="hidden items-center gap-1 text-sm sm:flex">
        <router-link
          to="/"
          class="rounded-md px-3 py-1.5 text-neutral-400 transition-colors hover:bg-accent/10 hover:text-accent"
          exact-active-class="!text-accent"
        >Overview</router-link>
        <router-link
          to="/blocks"
          class="rounded-md px-3 py-1.5 text-neutral-400 transition-colors hover:bg-accent/10 hover:text-accent"
          active-class="!text-accent"
        >Blocks</router-link>
      </nav>

      <div class="ml-auto flex min-w-0 items-center gap-3">
        <SearchBox />
        <router-link
          v-if="store.head"
          :to="`/block/${store.head}`"
          class="hidden shrink-0 items-center gap-1.5 rounded-md border border-accent/20 bg-accent/5 px-2.5 py-1 font-mono text-xs text-accent transition-colors hover:border-accent/50 md:flex"
          title="latest block"
        >
          <span class="muted">#</span>{{ fmtNum(store.head) }}
        </router-link>
      </div>
    </div>
  </header>
</template>
