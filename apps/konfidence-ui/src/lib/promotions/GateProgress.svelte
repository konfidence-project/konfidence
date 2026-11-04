<script lang="ts">
    import type { PromotionGate } from "./gates.ts";
    import { gateLabel, gateStateLabel, isPendingApproval, satisfiedCount } from "./gates.ts";

    interface Props { gates: readonly PromotionGate[]; }
    let { gates }: Props = $props();
    const satisfied = $derived(satisfiedCount(gates));
</script>

{#if gates.length}
    <span role="group" aria-label={`${satisfied} of ${gates.length} gates satisfied`} class="inline-flex flex-wrap items-center gap-3 text-compact text-content-secondary">
        <span class="inline-flex flex-wrap items-center gap-1 py-1">
            {#each gates as gate (gate.type)}
                <span role="img" aria-label={`${gateLabel(gate)}: ${gateStateLabel(gate)}`} title={`${gate.type}: ${gateStateLabel(gate)}`}
                    class={["size-2.5 shrink-0 rounded-[3px]", gate.state === "Satisfied" ? "bg-status-healthy-solid" : gate.state === "Error" ? "bg-status-error-solid" : "bg-[var(--track)]", isPendingApproval(gate) && "outline-2 outline-offset-2 outline-[var(--accent-primary)]"]}></span>
            {/each}
        </span>
        <span>{satisfied}/{gates.length} gates</span>
    </span>
{/if}
