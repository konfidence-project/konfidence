<script lang="ts">
    import { SelectableRow, StatusBadge } from "@konfidence/design-system/components";
    import type { Promotion } from "./promotions.ts";
    import { approvalGate, isActionable, statusLabel, statusTone } from "./promotions.ts";
    import PromotionActions from "./PromotionActions.svelte";
    import GateProgress from "./GateProgress.svelte";
    import { vectorLabel } from "./vector-label.ts";

    interface Props {
        promotion: Promotion;
    }
    let { promotion }: Props = $props();
</script>

<div data-row-id={promotion.id}>
<SelectableRow id={`promotion-row-${promotion.id}`} ariaLabel={`View promotion ${promotion.id}`} value={promotion.id} inset>
    <span class="flex min-w-0 flex-wrap items-center gap-x-5 gap-y-2">
        <StatusBadge status={statusTone(promotion)}>{statusLabel(promotion)}</StatusBadge>
        <span class="flex min-w-0 flex-1 basis-48 flex-col gap-1">
            <span title={promotion.vector || undefined} class="block max-w-full truncate font-mono font-semibold">{vectorLabel(promotion.vector) || "Vector not recorded"}</span>
            <span class="text-compact text-content-secondary">{promotion.id}</span>
            {#if approvalGate(promotion)?.setBy}
                <span class="text-meta text-content-secondary">Approval gate last updated by {approvalGate(promotion)?.setBy}</span>
            {/if}
        </span>
    </span>
    {#snippet actions()}
        {#if promotion.gates}<GateProgress gates={promotion.gates} />{/if}
        {#if isActionable(promotion)}<PromotionActions explain={false} />{/if}
    {/snippet}
</SelectableRow>
</div>
