<script lang="ts">
    import { Handle, Position } from "@xyflow/svelte";

    import StageCard from "$lib/components/stage/StageCard.svelte";

    type NodeProps = import("@xyflow/svelte").NodeProps;
    type Stage = import("$lib/stages").Stage;
    interface StageNodeData {
        stage: Stage;
    }

    const { data, selected = false } = $props<NodeProps & { data: StageNodeData }>();
</script>

<Handle type="target" position={Position.Left} />
<StageCard stage={data.stage} {selected} />
<Handle type="source" position={Position.Right} />
