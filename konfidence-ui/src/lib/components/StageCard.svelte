<script lang="ts">
    import type { Stage } from "$lib/stages.js";
    import type { StageCardVariant } from "$lib/stage-card-variants.js";
    import StageCardCustom from "$lib/components/stage-cards/StageCardCustom.svelte";
    import StageCardFiori from "$lib/components/stage-cards/StageCardFiori.svelte";
    import StageCardFioriHybrid from "$lib/components/stage-cards/StageCardFioriHybrid.svelte";

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
