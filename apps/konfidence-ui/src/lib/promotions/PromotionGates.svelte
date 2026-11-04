<script lang="ts">
    import type { PromotionGate } from "./gates.ts";
    import { gateLabel, gateStateLabel, gateTone, isPendingApproval, satisfiedCount } from "./gates.ts";

    interface Props { gates?: readonly PromotionGate[]; }
    let { gates }: Props = $props();
</script>

<section aria-label="Gates" class="flex flex-col gap-3">
    <h3 class="m-0 text-meta font-semibold uppercase tracking-wide text-content-tertiary">Gates</h3>
    {#if gates?.length}
        <ul class="m-0 flex list-none flex-col gap-2 p-0">
            {#each gates as gate (gate.type)}
                <li class={["flex min-w-0 items-start gap-3 rounded-[var(--radius-sm)] p-3", isPendingApproval(gate) && "bg-[var(--tag-amber-bg)]"]}>
                    <span aria-hidden="true" class={["flex size-7 shrink-0 items-center justify-center rounded-[var(--radius-sm)] font-semibold", gateTone(gate) === "healthy" ? "bg-status-healthy-bg text-status-healthy-fg" : gateTone(gate) === "error" ? "bg-status-error-bg text-status-error-fg" : "bg-status-queued-bg text-status-queued-fg"]}>{gate.state === "Satisfied" ? "✓" : gate.state === "Error" ? "!" : "−"}</span>
                    <div class="flex min-w-0 grow flex-col gap-1 [overflow-wrap:anywhere]">
                        <span class="flex flex-wrap items-center gap-2 text-compact font-medium">
                            <span title={gate.type}>{gateLabel(gate)}</span>
                            {#if isPendingApproval(gate)}<span class="rounded-[var(--radius-sm)] bg-status-warning-bg px-1.5 py-0.5 text-meta text-status-warning-fg">Approval required</span>{/if}
                        </span>
                        {#if gateLabel(gate) !== gate.type}<span class="font-mono text-meta text-content-tertiary">{gate.type}</span>{/if}
                        {#if gate.message}<span class="text-compact text-content-secondary">{gate.message}</span>{/if}
                        {#if gate.reason}<span class="text-meta text-content-tertiary">Reason: {gate.reason}</span>{/if}
                        {#if gate.setBy}<span class="text-meta text-content-tertiary">Recorded by {gate.setBy}</span>{/if}
                        {#if gate.setAt}<span class="text-meta text-content-tertiary">Recorded at {gate.setAt}</span>{/if}
                    </div>
                    <span class={["shrink-0 text-compact", gateTone(gate) === "healthy" ? "text-status-healthy-fg" : gateTone(gate) === "error" ? "text-status-error-fg" : "text-content-secondary"]}>{gateStateLabel(gate)}</span>
                </li>
            {/each}
        </ul>
        <p class="m-0 text-compact text-content-secondary">{satisfiedCount(gates)} of {gates.length} gates satisfied</p>
    {:else}
        <p class="m-0 text-compact text-content-secondary">{gates === undefined ? "Gate information is unavailable" : "No gates recorded"}</p>
    {/if}
</section>
