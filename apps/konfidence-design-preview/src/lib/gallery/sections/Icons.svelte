<script lang="ts">
    import { ICON_NAMES, Icon } from "@konfidence/design-system/components";
    import Section from "../Section.svelte";
</script>

<Section
    id="icons"
    title="Icons"
    subtitle="Curated SAP-Icons v5 subset (Apache-2.0), rendered at 16×16 viewBox via the DS `<Icon>` component."
    status="implemented"
>
    <p class="count">{ICON_NAMES.length} icons</p>
    <div class="grid">
        {#each ICON_NAMES as name (name)}
            <div class="cell">
                <Icon {name} />
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

    .cell :global(svg) {
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
