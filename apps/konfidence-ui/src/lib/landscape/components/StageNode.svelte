<script lang="ts">
    import type { Node, NodeProps } from "@xyflow/svelte";
    import { Handle, Position, useSvelteFlow } from "@xyflow/svelte";
    import type { Stage } from "#lib/konfidence-api/types.ts";
    import LandscapeStageCard from "#lib/landscape/components/LandscapeStageCard.svelte";
    import type { StageOverlayReference } from "#lib/landscape/stageOverlay.svelte.ts";

    type StageNode = Node<
        {
            stage: Stage;
            landscapeName: string;
            href: string;
            reference: StageOverlayReference;
            selected?: boolean;
            suppressFocusReveal?: boolean;
            onSelect?: (stage: Stage, trigger: HTMLElement) => void;
        },
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
            if (data.suppressFocusReveal) {
                return;
            }
            void fitView({ maxZoom: 1, nodes: [{ id }], padding: 0.2 });
        }
    };
    const openStage = (event: MouseEvent): void => {
        if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
            return;
        }
        event.preventDefault();
        data.onSelect?.(data.stage, event.currentTarget as HTMLElement);
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
    <!--
      Promotion edges connect stages left(dev)->right(prod). The handles are
      the required anchor points for those edges; they are presentational only
      (the graph is read-only) so they are hidden and inert.
    -->
    <Handle
        type="target"
        position={Position.Left}
        isConnectable={false}
        style="opacity: 0; pointer-events: none;"
    />
    <LandscapeStageCard {...data} onclick={openStage} />
    <Handle
        type="source"
        position={Position.Right}
        isConnectable={false}
        style="opacity: 0; pointer-events: none;"
    />
</div>
