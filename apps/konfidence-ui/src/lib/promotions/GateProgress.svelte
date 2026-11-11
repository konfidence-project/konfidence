<script lang="ts">
    import type { PromotionGate } from "./gates.ts";
    import { gateLabel, gateStateLabel, gateTone, isPendingApproval, satisfiedCount } from "./gates.ts";

    interface Props { gates: readonly PromotionGate[]; }
    let { gates }: Props = $props();
    const satisfied = $derived(satisfiedCount(gates));
</script>

{#if gates.length}
    <span role="group" aria-label={`${satisfied} of ${gates.length} gates satisfied`} class="inline-flex flex-wrap items-center gap-3 text-compact text-content-secondary">
        <span class="inline-flex flex-wrap items-center gap-1 py-1">
            {#each gates as gate (gate.type)}
                {@const tone = gateTone(gate)}
                {@const label = gateLabel(gate)}
                {@const stateLabel = gateStateLabel(gate)}
                {@const pendingApproval = isPendingApproval(gate)}
                <span role="img" aria-label={`${label}: ${stateLabel}`} title={`${gate.type}: ${stateLabel}`}
                    class={["size-2.5 shrink-0 rounded-[3px]", tone === "healthy" ? "bg-status-healthy-solid" : tone === "error" ? "bg-status-error-solid" : "bg-[var(--track)]", pendingApproval && "outline-2 outline-offset-2 outline-[var(--accent-primary)]"]}></span>
            {/each}
        </span>
        <span>{satisfied}/{gates.length} gates</span>
    </span>
{/if}
