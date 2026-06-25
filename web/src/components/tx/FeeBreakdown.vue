<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { RawReceipt, TxFull } from "../../types";
import { api } from "../../api";
import { computeFees, formatEthExact, formatGweiExact, pct, trimDecimals } from "../../decode";
import { fmtNum } from "../../format";

const props = defineProps<{ tx: TxFull; receipt: RawReceipt }>();

const blockBaseFee = ref<number | null>(null);
const blockLoaded = ref(false);

onMounted(async () => {
  if (props.tx.blockNumber != null) {
    try {
      const res = await api.block(props.tx.blockNumber);
      blockBaseFee.value = res.block.baseFeeGwei;
    } catch { /* fall back to derived base fee */ }
  }
  blockLoaded.value = true;
});

const fees = computed(() => computeFees(props.tx, props.receipt, blockBaseFee.value));

// Bar geometry: the track spans maxCost when known, else totalFee.
const track = computed(() => {
  const f = fees.value;
  const whole = f.maxCostWei && f.maxCostWei > f.totalFeeWei ? f.maxCostWei : f.totalFeeWei;
  return {
    burnedPct: f.burnedWei != null ? pct(f.burnedWei, whole) : null,
    tipPct: f.tipWei != null ? pct(f.tipWei, whole) : pct(f.totalFeeWei, whole),
    savedPct: f.savedWei != null ? pct(f.savedWei, whole) : 0,
  };
});

const gasUtilPct = computed(() => {
  const f = fees.value;
  return f.gasLimit > 0n ? pct(f.gasUsed, f.gasLimit) : 0;
});

const eth = (v: bigint) => trimDecimals(formatEthExact(v), 8);
const gwei = (v: bigint) => trimDecimals(formatGweiExact(v), 4);
</script>

<template>
  <section class="border-t border-accent/15 pt-4">
    <header class="mb-3 flex items-center gap-2">
      <h2 class="label">Fee breakdown</h2>
      <span v-if="fees.approximate" class="rounded border border-amber-500/30 bg-amber-500/10 px-1.5 py-0.5 font-mono text-[10px] text-amber-400" title="node omitted effectiveGasPrice; reconstructed from tx fields">approx</span>
    </header>

    <!-- stacked bar -->
    <div class="space-y-1.5">
      <div class="flex h-4 w-full overflow-hidden rounded border border-accent/15 bg-bg/60">
        <div
          v-if="track.burnedPct != null"
          class="h-full bg-eth/70 transition-all duration-700"
          :style="{ width: `${track.burnedPct}%` }"
          :title="`burned: ${fees.burnedWei != null ? eth(fees.burnedWei) : '?'} ETH`"
        />
        <div
          class="h-full bg-accent/80 transition-all duration-700"
          :style="{ width: `${track.tipPct}%` }"
          :title="`tip: ${fees.tipWei != null ? eth(fees.tipWei) : eth(fees.totalFeeWei)} ETH`"
        />
        <div
          v-if="track.savedPct > 0"
          class="saved-stripes h-full"
          :style="{ width: `${track.savedPct}%` }"
          title="unspent vs max fee cap"
        />
      </div>
      <div class="flex flex-wrap gap-x-4 gap-y-1 font-mono text-[10px] muted">
        <span v-if="fees.burnedWei != null" class="flex items-center gap-1.5">
          <span class="h-2 w-2 rounded-sm bg-eth/70" /> burned (base fee)
        </span>
        <span class="flex items-center gap-1.5">
          <span class="h-2 w-2 rounded-sm bg-accent/80" /> {{ fees.tipWei != null ? "tip (priority fee)" : "total fee" }}
        </span>
        <span v-if="track.savedPct > 0" class="flex items-center gap-1.5">
          <span class="saved-stripes h-2 w-2 rounded-sm" /> saved vs max
        </span>
      </div>
    </div>

    <!-- rows -->
    <dl class="mt-4 divide-y divide-accent/10 font-mono text-xs">
      <div class="flex items-baseline justify-between py-2">
        <dt class="muted">Total fee</dt>
        <dd class="tabular font-bold text-neutral-100">{{ eth(fees.totalFeeWei) }} <span class="muted font-normal">ETH</span></dd>
      </div>
      <div class="flex items-baseline justify-between py-2">
        <dt class="muted">Effective gas price</dt>
        <dd class="tabular">{{ gwei(fees.effectiveGasPriceWei) }} <span class="muted">gwei</span></dd>
      </div>
      <div v-if="fees.baseFeeWei != null" class="flex items-baseline justify-between py-2">
        <dt class="muted">
          Base fee
          <span v-if="fees.baseFeeSource === 'derived'" class="text-amber-400/70" title="derived from effectiveGasPrice − tip; exact only when the tip wasn't clamped by the max fee">~</span>
        </dt>
        <dd class="tabular">{{ gwei(fees.baseFeeWei) }} <span class="muted">gwei</span></dd>
      </div>
      <div v-if="fees.burnedWei != null" class="flex items-baseline justify-between py-2">
        <dt class="muted">Burned</dt>
        <dd class="tabular text-eth-muted">{{ eth(fees.burnedWei) }} <span class="muted">ETH</span></dd>
      </div>
      <div v-if="fees.tipWei != null" class="flex items-baseline justify-between py-2">
        <dt class="muted">Validator tip</dt>
        <dd class="tabular text-accent/90">{{ eth(fees.tipWei) }} <span class="muted">ETH</span></dd>
      </div>
      <div v-if="fees.savedWei != null && fees.maxCostWei != null" class="flex items-baseline justify-between py-2">
        <dt class="muted">Saved vs max cost</dt>
        <dd class="tabular text-green-400/90">{{ eth(fees.savedWei) }} <span class="muted">of {{ eth(fees.maxCostWei) }} ETH</span></dd>
      </div>
      <div class="flex items-baseline justify-between py-2">
        <dt class="muted">Gas used</dt>
        <dd class="tabular">
          {{ fmtNum(Number(fees.gasUsed)) }} <span class="muted">/ {{ fmtNum(Number(fees.gasLimit)) }} ({{ gasUtilPct.toFixed(1) }}%)</span>
        </dd>
      </div>
      <div v-if="fees.blobFeeWei != null" class="flex items-baseline justify-between py-2">
        <dt class="muted">Blob fee</dt>
        <dd class="tabular text-eth-muted">{{ eth(fees.blobFeeWei) }} <span class="muted">ETH</span></dd>
      </div>
    </dl>
  </section>
</template>

<style scoped>
.saved-stripes {
  background: repeating-linear-gradient(
    -45deg,
    rgba(74, 222, 128, 0.25) 0 4px,
    rgba(74, 222, 128, 0.07) 4px 8px
  );
}
</style>
