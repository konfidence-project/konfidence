<script lang="ts">
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/AllIcons.js";
    import Section from "../Section.svelte";

    /**
     * Curated SAP-Icons subset actually referenced by `@konfidence/design-system`
     * and `@konfidence/konfidence-ui`. Icons are pulled directly from the
     * `@ui5/webcomponents-icons` package — there is no DS-level `Icon`
     * component anymore; components inline `<ui5-icon>` on demand.
     */
    const ICON_NAMES = [
        "add",
        "bell",
        "chain-link",
        "decline",
        "edit",
        "error",
        "filter",
        "grid",
        "log",
        "menu2",
        "product",
        "search",
        "settings",
        "slim-arrow-down",
        "sys-help",
        "sys-monitor",
        "upload",
        "user-edit",
    ] as const;
</script>

<Section
    id="icons"
    title="Icons"
    subtitle="Curated SAP-Icons v5 subset (Apache-2.0), rendered on demand via the `<ui5-icon>` web component (no DS-level wrapper)."
    status="implemented"
>
    <p class="count">{ICON_NAMES.length} icons in the curated subset</p>
    <div class="grid">
        {#each ICON_NAMES as name (name)}
            <div class="cell">
                <ui5-icon class="icon" {name}></ui5-icon>
                <code>{name}</code>
            </div>
        {/each}
    </div>
</Section>

<style>
    .count {
        margin: 0 0 12px;
        font-size: var(--text-meta);
        color: var(--text-secondary);
    }

    .grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        gap: 12px;
    }

    .cell {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 6px;
        padding: 10px 8px;
        border: 1px solid var(--border-subtle);
        border-radius: 8px;
        background: var(--surface-default);
    }

    .cell :global(ui5-icon.icon) {
        width: 24px;
        height: 24px;
        color: var(--text-primary);
    }

    .cell code {
        font-family: var(--font-mono);
        font-size: 11px;
        color: var(--text-tertiary, var(--text-secondary));
        text-align: center;
    }
</style>
