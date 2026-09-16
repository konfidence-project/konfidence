<script lang="ts">
    import type { Node, NodeProps } from "@xyflow/svelte";
    import { useSvelteFlow } from "@xyflow/svelte";
    import type { Stage } from "$lib/landscape/landscapeApi";
    import LandscapeStageCard from "$lib/landscape/components/LandscapeStageCard.svelte";

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

<!--
  Wrapper carries no `.nodrag` / `.nopan` opt-outs. Node repositioning
  is already disabled at the SvelteFlow level (`nodesDraggable={false}`
  in LandscapeFlow), and dropping `.nopan` lets touches/drags that land
  on a card initiate a canvas pan — critical for mobile viewports where
  cards cover most of the pane. Short taps still fire the underlying
  `<a>` link because xyflow uses a small drag threshold before it
  commits to a pan.
-->
<div onfocusin={revealFocusedCard}>
    <LandscapeStageCard {...data} />
</div>
