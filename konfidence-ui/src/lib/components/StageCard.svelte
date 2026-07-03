<script lang="ts" module>
    export type StageCardVariant = "fiori" | "fiori-mockup" | "custom";

    export const STAGE_CARD_VARIANTS: {
        id: StageCardVariant;
        label: string;
        description: string;
    }[] = [
        {
            id: "fiori",
            label: "Fiori",
            description: "Pure UI5 Web Components; inherits SAP theming.",
        },
        {
            id: "fiori-mockup",
            label: "Fiori · Mockup",
            description:
                "UI5 primitives (Card, Menu, Tag, Icon) laid out to match the Konfidence mockup.",
        },
        {
            id: "custom",
            label: "Custom",
            description: "No UI5 wc — hand-rolled markup, closest to the mockup.",
        },
    ];
</script>

<script lang="ts">
    import type { Stage } from "$lib/stages.js";
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
