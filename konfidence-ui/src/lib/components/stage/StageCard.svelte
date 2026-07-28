<script lang="ts">
    import StageCardFiori from "./cards/StageCardFiori.svelte";
    import type { Stage } from "$lib/stages.js";

    const { stage, selected = false } = $props<{ stage: Stage; selected?: boolean }>();
</script>

<StageCardFiori {stage} {selected} />
