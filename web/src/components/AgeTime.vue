<script setup lang="ts">
import { computed } from "vue";
import { fmtAge } from "../format";
import { now } from "../ticker";

const props = defineProps<{ unixMs: number }>();
const age = computed(() => fmtAge(props.unixMs, now.value));
</script>

<template>
  <time class="tabular">{{ age }}</time>
</template>
