<script lang="ts">
    import "@ui5/webcomponents/dist/Card.js";
    import "@ui5/webcomponents/dist/CardHeader.js";
    import "@ui5/webcomponents/dist/Label.js";
    import "@ui5/webcomponents/dist/Option.js";
    import "@ui5/webcomponents/dist/Select.js";
    import type { SelectChangeEventDetail } from "@ui5/webcomponents/dist/Select.js";

    import { STAGE_CARD_VARIANTS } from "$lib/components/StageCard.svelte";
    import { getStageCardVariantPreference } from "$lib/stores/stage-card-variant.svelte";

    let { id = "stage-card-variant-picker" } = $props<{ id?: string }>();

    const stageCardVariant = getStageCardVariantPreference();

    function handleVariantChange(event: CustomEvent<SelectChangeEventDetail>) {
        stageCardVariant.select(event.detail.selectedOption.value ?? "");
    }
</script>

<ui5-card accessible-name="Stage card settings">
    <ui5-card-header
        slot="header"
        title-text="Stage Cards"
        subtitle-text="Choose how stages are rendered across the app"
    ></ui5-card-header>

    <div class="content">
        <div class="label-group">
            <ui5-label for={id} required>Card style</ui5-label>
            <p>The selected variant is used in the landscape view.</p>
        </div>

        <ui5-select
            {id}
            accessible-name="Stage card style"
            value={stageCardVariant.selected}
            onui5-change={handleVariantChange}
        >
            {#each STAGE_CARD_VARIANTS as option (option.id)}
                <ui5-option value={option.id}>{option.label}</ui5-option>
            {/each}
        </ui5-select>
    </div>
</ui5-card>

<style>
    .content {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1.5rem;
        padding: 1rem 1.25rem 1.25rem;
    }

    .label-group {
        display: grid;
        gap: 0.25rem;
    }

    p {
        margin: 0;
        color: var(--sapContent_LabelColor);
        font-size: var(--sapFontSmallSize);
    }

    ui5-select {
        min-width: 14rem;
    }

    @media (max-width: 42rem) {
        .content {
            align-items: stretch;
            flex-direction: column;
        }

        ui5-select {
            width: 100%;
        }
    }
</style>
