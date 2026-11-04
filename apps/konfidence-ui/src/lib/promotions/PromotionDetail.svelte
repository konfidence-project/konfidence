<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import type { Landscape } from "#lib/konfidence-api/types.ts";
    import type { Promotion } from "./promotions.ts";
    import { approvalGate, referenceLabel, statusLabel, statusTone } from "./promotions.ts";
    import PromotionGates from "./PromotionGates.svelte";

    interface Props { promotion: Promotion; landscapes: readonly Landscape[]; }
    let { promotion, landscapes }: Props = $props();
    const approval = $derived(approvalGate(promotion));
    const approvalLabels: Record<string, string> = { Error: "Approval gate error", Pending: "Awaiting approval", Satisfied: "Approved" };
    const approvalLabel = $derived(approval ? approvalLabels[approval.state] ?? approval.state ?? "Unknown" : "No explicit-approval gate recorded");
    const fieldClass = "min-w-0 text-compact text-content-primary [overflow-wrap:anywhere]";
</script>

<div class="flex flex-col gap-5">
    <section aria-label="Promotion flow" class="flex min-w-0 flex-wrap items-center justify-between gap-3 rounded-[var(--card-radius)] border border-outline-subtle bg-surface-sunken p-4">
        <div class="flex min-w-0 flex-1 flex-col gap-1 [overflow-wrap:anywhere]">
            <span class="font-semibold">{promotion.source.name}</span>
            <span class="text-compact text-content-secondary">Source · {promotion.source.kind}</span>
        </div>
        <span aria-hidden="true" class="text-content-secondary">→</span>
        <div class="flex min-w-0 flex-1 flex-col gap-1 text-right [overflow-wrap:anywhere]">
            <span class="font-semibold">{promotion.target.name}</span>
            <span class="text-compact text-content-secondary">Target · {promotion.target.kind}</span>
        </div>
    </section>
    <section aria-label="Promotion information" class="flex flex-col gap-3">
    <h3 class="m-0 text-meta font-semibold uppercase tracking-wide text-content-tertiary">Info</h3>
    <dl class="grid grid-cols-[minmax(0,7rem)_minmax(0,1fr)] gap-x-4 gap-y-3 text-compact">
        <dt class="text-content-tertiary">Promotion ID</dt><dd class={fieldClass}>{promotion.id}</dd>
        <dt class="text-content-tertiary">Status</dt><dd class={fieldClass}><StatusBadge status={statusTone(promotion)}>{statusLabel(promotion)}</StatusBadge></dd>
        <dt class="text-content-tertiary">Vector</dt><dd class={[fieldClass, "font-mono"]}>{promotion.vector || "Not recorded"}</dd>
        <dt class="text-content-tertiary">Source</dt><dd class={fieldClass}>{referenceLabel(promotion.source, landscapes)}</dd>
        <dt class="text-content-tertiary">Target</dt><dd class={fieldClass}>{referenceLabel(promotion.target, landscapes)}</dd>
        <dt class="text-content-tertiary">Approval</dt><dd class={fieldClass}>{approvalLabel}</dd>
        {#if approval?.setBy}<dt class="text-content-tertiary">Recorded by</dt><dd class={fieldClass}>{approval.setBy}</dd>{/if}
        {#if approval?.setAt}<dt class="text-content-tertiary">Recorded at</dt><dd class={fieldClass}>{approval.setAt}</dd>{/if}
        {#if approval?.message}<dt class="text-content-tertiary">Approval message</dt><dd class={fieldClass}>{approval.message}</dd>{/if}
    </dl>
    </section>
    <PromotionGates gates={promotion.gates} />
    <p class="m-0 text-compact text-content-secondary">Source and target reflect this promotion, not the configuration’s current flow.</p>
    {#if promotion.status === "Succeeded"}
        <p class="m-0 text-compact text-content-secondary">Succeeded means the target stage was updated. It does not confirm deployment rollout or activation.</p>
    {/if}
</div>
