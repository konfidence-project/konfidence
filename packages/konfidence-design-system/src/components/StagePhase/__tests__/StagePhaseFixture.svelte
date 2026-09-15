<script lang="ts">
    import StagePhase from "../StagePhase.svelte";
    import type { StagePhaseItem, StagePhaseSize } from "../types.js";

    interface Props {
        phases: StagePhaseItem[];
        size?: StagePhaseSize;
        ariaLabel?: string;
    }

    let { phases, size, ariaLabel }: Props = $props();
</script>

<div data-testid="stage-phase-root">
    <StagePhase {phases} {size} {ariaLabel} />
</div>
