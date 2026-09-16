<script lang="ts">
    import StageCard from "../StageCard.svelte";
    import type { StagePhaseItem } from "../../StagePhase/types.js";
    import type { StageCardStatusRole } from "../types.js";

    interface Props {
        title: string;
        href: string;
        ariaLabel: string;
        statusRole: StageCardStatusRole;
        phases: StagePhaseItem[];
        phaseAriaLabel: string;
        targetVector?: string;
        activeVersionText: string;
        live?: boolean;
        selected?: boolean;
    }

    let props: Props = $props();
</script>

<StageCard {...props} />
