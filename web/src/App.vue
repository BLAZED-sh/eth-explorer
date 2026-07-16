<script setup lang="ts">
import NavBar from "./components/NavBar.vue";
</script>

<template>
  <div class="min-h-screen bg-bg">
    <!-- atmosphere layers -->
    <div class="bg-grid pointer-events-none fixed inset-0" aria-hidden="true" />
    <div class="bg-grain pointer-events-none fixed inset-0" aria-hidden="true" />
    <div class="bg-vignette fixed inset-0" aria-hidden="true" />

    <div class="relative z-10">
      <NavBar />
      <main class="mx-auto w-full max-w-7xl px-4 pb-16 pt-6 sm:px-6">
        <router-view />
      </main>
      <footer class="border-t border-accent/10 py-6 text-center text-xs muted">
        <span class="font-mono">blazed explorer</span> — live Ethereum mempool ·
        powered by
        <a
          href="https://blazed.sh/?utm_source=explorer&utm_medium=referral&utm_campaign=footer"
          target="_blank"
          rel="noopener"
          class="text-accent transition-colors hover:text-accent/80"
        >blazed.sh</a>
      </footer>
    </div>
  </div>
</template>
