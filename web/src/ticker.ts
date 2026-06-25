import { ref } from "vue";

// One shared 1s ticker for every relative-time display on the page (AgeTime,
// TxLifecycle, ...). Module-scoped, so every importer reads the same ref and
// there is a single setInterval for the whole app.
export const now = ref(Date.now());
setInterval(() => (now.value = Date.now()), 1000);
