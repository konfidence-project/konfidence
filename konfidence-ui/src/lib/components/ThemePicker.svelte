<script lang="ts">
    import "@ui5/webcomponents/dist/Card.js";
    import "@ui5/webcomponents/dist/CardHeader.js";
    import "@ui5/webcomponents/dist/Label.js";
    import "@ui5/webcomponents/dist/Option.js";
    import "@ui5/webcomponents/dist/Select.js";
    import type { SelectChangeEventDetail } from "@ui5/webcomponents/dist/Select.js";

    import {
        selectTheme,
        themePreference,
        themes,
    } from "$lib/stores/theme.svelte";

    let { id = "theme-picker" } = $props<{ id?: string }>();

    function handleThemeChange(event: CustomEvent<SelectChangeEventDetail>) {
        selectTheme(event.detail.selectedOption.value ?? "");
    }
</script>

<ui5-card accessible-name="Theme settings">
    <ui5-card-header
        slot="header"
        title-text="Theme"
        subtitle-text="Choose your preferred theme"
    ></ui5-card-header>

    <div class="content">
        <ui5-label for={id} required>Theme</ui5-label>

        <ui5-select
            {id}
            accessible-name="Theme"
            value={themePreference.selected}
            onui5-change={handleThemeChange}
        >
            {#each themes as option (option.id)}
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
