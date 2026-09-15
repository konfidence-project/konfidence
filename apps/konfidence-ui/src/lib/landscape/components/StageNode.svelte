<script lang="ts">
    import type { Node, NodeProps } from "@xyflow/svelte";
    import { useSvelteFlow } from "@xyflow/svelte";
    import type { Stage } from "$lib/landscape/landscapeApi";
    import StageCard from "$lib/landscape/components/StageCard.svelte";

    type StageNode = Node<
        { stage: Stage; landscapeName: string; category: string; href: string },
        "stage"
    >;
    let { data, id }: NodeProps<StageNode> = $props();
    const { fitView } = useSvelteFlow();

    // Keyboard focus can reach cards outside the panned viewport; bring them into view.
    const revealFocusedCard = (event: FocusEvent): void => {
        if (
            event.target instanceof HTMLElement &&
            event.target.matches(":focus-visible")
        ) {
            void fitView({ maxZoom: 1, nodes: [{ id }], padding: 0.2 });
        }
    };
</script>

<div class="nodrag nopan" onfocusin={revealFocusedCard}>
    <StageCard {...data} />
</div>
