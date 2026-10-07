<script lang="ts">
    import type { Node, NodeProps } from "@xyflow/svelte";
    import { StageCardPlaceholder } from "@konfidence/design-system/components";

    type LandscapeEmptyNode = Node<{ landscapeName: string }, "landscape-empty">;
    let { data }: NodeProps<LandscapeEmptyNode> = $props();
</script>

<StageCardPlaceholder
    title={data.landscapeName}
    message="No stages yet"
    ariaLabel={`Landscape ${data.landscapeName} has no stages yet`}
/>
