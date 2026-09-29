<script lang="ts">
    import type { Node, NodeProps } from "@xyflow/svelte";
    import LandscapeEmptyCard from "$lib/landscape/components/LandscapeEmptyCard.svelte";

    type LandscapeEmptyNode = Node<{ landscapeName: string }, "landscape-empty">;
    let { data }: NodeProps<LandscapeEmptyNode> = $props();
</script>

<LandscapeEmptyCard {...data} />
