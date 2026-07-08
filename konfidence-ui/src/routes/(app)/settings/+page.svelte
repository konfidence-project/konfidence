<script lang="ts">
    import "@ui5/webcomponents/dist/Title.js";
    import "@ui5/webcomponents-fiori/dist/Page.js";

    import StageCardVariantPicker from "$lib/components/stage/StageCardVariantPicker.svelte";
    import ThemePicker from "$lib/components/ThemePicker.svelte";
</script>

<ui5-page background-design="Solid">
    <header slot="header" class="page-header">
        <ui5-title id="settings-title" level="H1" size="H2">Settings</ui5-title>
    </header>

    <div class="page-content" aria-labelledby="settings-title">
        <ThemePicker id="theme-picker" />
        <StageCardVariantPicker id="stage-card-variant-picker" />
    </div>
</ui5-page>

<style>
    ui5-page {
        height: 100%;
    }

    .page-header {
        padding: 1.5rem clamp(1rem, 3vw, 2rem) 1rem;
        background: var(--sapBackgroundColor);
        border-bottom: 1px solid var(--sapList_BorderColor);
        box-sizing: border-box;
    }

    .page-content {
        display: grid;
        gap: 1rem;
        max-width: 56rem;
        padding: 1.5rem clamp(1rem, 3vw, 2rem);
        box-sizing: border-box;
    }
</style>
