<script lang="ts">
    import type { Stage } from "$lib/stages.js";
    import StageCardFiori from "./cards/StageCardFiori.svelte";

    const { stage, selected = false } = $props<{ stage: Stage; selected?: boolean }>();
</script>

<StageCardFiori {stage} {selected} />
