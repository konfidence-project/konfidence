<script lang="ts">
    import type { Stage } from "$lib/stages.js";
    import StageCardCustom from "./cards/StageCardCustom.svelte";
    import StageCardFiori from "./cards/StageCardFiori.svelte";
    import StageCardFioriHybrid from "./cards/StageCardFioriHybrid.svelte";
    import type { StageCardVariant } from "./variants.js";

    const {
        stage,
        variant = "custom",
        selected = false,
    } = $props<{
        stage: Stage;
        variant?: StageCardVariant;
        selected?: boolean;
    }>();
</script>

{#if variant === "fiori"}
    <StageCardFiori {stage} />
{:else if variant === "fiori-mockup"}
    <StageCardFioriHybrid {stage} {selected} />
{:else}
    <StageCardCustom {stage} {selected} />
{/if}
