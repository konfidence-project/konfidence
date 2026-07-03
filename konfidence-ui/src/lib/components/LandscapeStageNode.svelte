<script lang="ts">
    import { Handle, Position, type NodeProps } from "@xyflow/svelte";

    import StageCard, {
        type StageCardVariant,
    } from "$lib/components/StageCard.svelte";
    import type { Stage } from "$lib/stages.js";

    type StageNodeData = {
        stage: Stage;
        variant: StageCardVariant;
    };

    const { data, selected = false } = $props<NodeProps & { data: StageNodeData }>();
</script>

<Handle type="target" position={Position.Left} />
<StageCard stage={data.stage} variant={data.variant} {selected} />
<Handle type="source" position={Position.Right} />

<style>
    :global(.svelte-flow__handle) {
        width: 0.6rem;
        height: 0.6rem;
        border-color: var(--sapContent_ForegroundBorderColor);
        background: var(--sapHighlightColor);
    }
</style>
