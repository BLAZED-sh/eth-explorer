<script setup lang="ts">
// Client-side EVM bytecode analysis powered by `sevm` (loaded lazily so it
// never enters the main bundle). We surface what a node + static analysis can
// know for sure: detected ERC standards, the Solidity metadata fingerprint,
// emitted event topics, and the opcode disassembly — segmented into the public
// function blocks (entry points marked) so each can be collapsed.
// No external 4byte/source lookups — function names come from our own
// well-known selector map, falling back to the raw selector.
import { onMounted, ref, shallowRef } from "vue";
import { methodLabel, shortHash } from "../../format";

const props = defineProps<{ bytecode: string }>();

interface Op {
  pc: number;
  mnemonic: string;
  data: string | null;
}
interface Block {
  // null entry = the dispatcher / preamble before the first function
  entry: { sig: string; name: string } | null;
  startPc: number;
  ops: Op[];
}
interface Analysis {
  ercs: string[];
  solc: string;
  metaProtocol: string;
  metaHash: string;
  events: string[]; // topic0 hashes
  blocks: Block[];
  opCount: number;
}

const loading = ref(true);
const error = ref("");
const analysis = shallowRef<Analysis | null>(null);
const open = ref<Record<number, boolean>>({});

// EIP standards worth flagging, in the order we show them.
const ERC_CHECKS = ["ERC20", "ERC721", "ERC1155", "ERC165", "ERC173"] as const;

function fmtPc(n: number): string {
  return "0x" + n.toString(16).padStart(4, "0");
}

onMounted(async () => {
  try {
    const sevm = await import("sevm");
    // Symbolic execution can be a little heavy; yield a frame first so the
    // surrounding page paints before we crunch.
    await new Promise((r) => requestAnimationFrame(() => r(null)));
    const c: any = new sevm.Contract(props.bytecode);

    const ercs: string[] = [];
    for (const id of ERC_CHECKS) {
      try {
        if (c.isERC(id)) ercs.push(id);
      } catch {
        /* isERC may bail on unusual bytecode — skip */
      }
    }

    // Disassembly: opcodes() is neither pc-ordered nor unique, so sort and
    // dedupe by program counter to get a clean linear listing.
    const byPc = new Map<number, Op>();
    for (const o of c.opcodes() as any[]) {
      if (!byPc.has(o.pc)) {
        byPc.set(o.pc, { pc: o.pc, mnemonic: o.mnemonic, data: o.hexData?.() ?? null });
      }
    }
    const ops = [...byPc.values()].sort((a, b) => a.pc - b.pc);

    // Function entry points → pc of the JUMPDEST where each public/external
    // function body begins.
    const entryByPc = new Map<number, { sig: string; name: string }>();
    for (const [sel, v] of c.functionBranches as Map<string, { pc: number }>) {
      const sig = "0x" + sel;
      const known = methodLabel(sig, 1);
      entryByPc.set(v.pc, { sig, name: known !== sig ? known : "" });
    }

    // Segment the linear listing at each entry point. Everything before the
    // first entry is the dispatcher/preamble (entry = null).
    const blocks: Block[] = [];
    let cur: Block | null = null;
    for (const op of ops) {
      const entry = entryByPc.get(op.pc);
      if (entry || !cur) {
        cur = { entry: entry ?? null, startPc: op.pc, ops: [] };
        blocks.push(cur);
      }
      cur.ops.push(op);
    }

    const m = c.metadata;
    analysis.value = {
      ercs,
      solc: m?.solc ?? "",
      metaProtocol: m?.protocol ?? "",
      metaHash: m?.hash ?? "",
      events: Object.keys(c.events ?? {}),
      blocks,
      opCount: ops.length,
    };
    // Default: dispatcher open, function blocks collapsed.
    const init: Record<number, boolean> = {};
    blocks.forEach((b, i) => (init[i] = b.entry === null));
    open.value = init;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "failed to analyze bytecode";
  } finally {
    loading.value = false;
  }
});

function toggle(i: number) {
  open.value = { ...open.value, [i]: !open.value[i] };
}
</script>

<template>
  <div v-if="loading" class="space-y-2">
    <div class="skeleton h-5 w-40" />
    <div class="skeleton h-24" />
  </div>

  <div v-else-if="error" class="font-mono text-xs text-amber-400/90">
    couldn’t analyze bytecode — {{ error }}
  </div>

  <div v-else-if="analysis" class="space-y-5">
    <!-- standards + compiler fingerprint -->
    <div class="flex flex-wrap items-center gap-2">
      <span
        v-for="erc in analysis.ercs"
        :key="erc"
        class="rounded border border-accent/40 bg-accent/10 px-2 py-0.5 font-mono text-[11px] font-semibold uppercase tracking-wider text-accent"
      >{{ erc }}</span>
      <span v-if="!analysis.ercs.length" class="font-mono text-[11px] muted">no standard interface detected</span>
      <span class="ml-auto font-mono text-[11px] muted">
        <template v-if="analysis.solc">solc {{ analysis.solc }}</template>
        <template v-else-if="analysis.metaProtocol">{{ analysis.metaProtocol }}</template>
        <template v-else>no metadata</template>
        <span v-if="analysis.metaHash" class="ml-1.5" :title="analysis.metaHash">· {{ shortHash(analysis.metaHash, 6, 4) }}</span>
      </span>
    </div>

    <!-- event topics -->
    <div v-if="analysis.events.length">
      <div class="label mb-2">Events <span class="muted">· {{ analysis.events.length }}</span></div>
      <div class="flex flex-wrap gap-1.5">
        <code
          v-for="ev in analysis.events"
          :key="ev"
          class="rounded border border-accent/10 bg-bg/60 px-2 py-0.5 font-mono text-[10px] text-neutral-400"
          :title="ev"
        >{{ shortHash(ev, 8, 6) }}</code>
      </div>
    </div>

    <!-- disassembly, segmented into collapsible function blocks -->
    <div>
      <div class="label mb-2">Disassembly <span class="muted">· {{ analysis.opCount }} ops</span></div>
      <div class="overflow-hidden rounded-lg border border-accent/10 bg-bg/70">
        <div v-for="(b, i) in analysis.blocks" :key="i" class="border-b border-accent/10 last:border-b-0">
          <!-- block header / entry-point marker -->
          <button
            type="button"
            class="flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors hover:bg-accent/[0.04] active:scale-[0.997]"
            @click="toggle(i)"
          >
            <span class="w-3 shrink-0 select-none font-mono text-[10px] text-accent/60">{{ open[i] ? "▾" : "▸" }}</span>
            <template v-if="b.entry">
              <span class="shrink-0 rounded bg-accent/15 px-1.5 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-wider text-accent/90">fn</span>
              <code class="shrink-0 font-mono text-[11px] text-accent/70">{{ b.entry.sig }}</code>
              <span v-if="b.entry.name" class="truncate font-mono text-[12px] text-neutral-200">{{ b.entry.name }}</span>
            </template>
            <template v-else>
              <span class="font-mono text-[11px] font-semibold uppercase tracking-wider text-neutral-400">
                {{ analysis.blocks.length > 1 ? "dispatcher" : "code" }}
              </span>
            </template>
            <span class="ml-auto shrink-0 select-none font-mono text-[10px] muted">{{ fmtPc(b.startPc) }} · {{ b.ops.length }} ops</span>
          </button>
          <!-- opcodes -->
          <div v-if="open[i]" class="border-t border-accent/[0.06]">
            <div
              v-for="(op, j) in b.ops"
              :key="op.pc"
              class="flex items-center gap-3 px-3 py-0.5"
              :class="j % 2 === 1 && 'bg-accent/[0.03]'"
            >
              <span class="w-14 shrink-0 select-none font-mono text-[10px] muted">{{ fmtPc(op.pc) }}</span>
              <span class="w-28 shrink-0 font-mono text-[11px] font-semibold text-accent/80">{{ op.mnemonic }}</span>
              <code v-if="op.data" class="break-all font-mono text-[11px] text-neutral-400">0x{{ op.data }}</code>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- raw bytecode -->
    <details class="group">
      <summary class="cursor-pointer select-none font-mono text-[11px] text-accent/70 transition-colors hover:text-accent">
        raw bytecode
      </summary>
      <code class="mt-2 block max-h-64 overflow-auto break-all rounded-lg border border-accent/10 bg-bg/70 p-3 font-mono text-[10px] leading-relaxed text-neutral-500">{{ props.bytecode }}</code>
    </details>
  </div>
</template>
