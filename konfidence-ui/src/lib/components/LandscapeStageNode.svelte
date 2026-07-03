<script lang="ts">
    import { Handle, Position } from "@xyflow/svelte";

    import StageCard from "$lib/components/StageCard.svelte";

    type NodeProps = import("@xyflow/svelte").NodeProps;
    type Stage = import("$lib/stages.js").Stage;
    type StageCardVariant = import("$lib/stage-card-variants.js").StageCardVariant;

    interface StageNodeData {
        stage: Stage;
        variant: StageCardVariant;
    }

    const { data, selected = false } = $props<NodeProps & { data: StageNodeData }>();
</script>

<Handle type="target" position={Position.Left} />
<StageCard stage={data.stage} variant={data.variant} {selected} />
<Handle type="source" position={Position.Right} />

<style>
    :global(.svelte-flow__handle) {
        width: 0.6rem;
        height: 0.6rem;
        border: 0;
        background: transparent;
        opacity: 0;
        pointer-events: none;
    }
</style>
