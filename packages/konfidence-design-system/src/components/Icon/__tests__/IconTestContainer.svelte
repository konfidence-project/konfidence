<script lang="ts">
    /**
     * Renders the entire icon set in a grid so a single screenshot pins
     * every icon's rendered SVG at once. Cheaper than 48 individual
     * snapshots and still detects any single-icon regression.
     */
    import Icon from "../Icon.svelte";
    import { ICON_NAMES } from "../icons.js";
</script>

<div
    data-testid="icon-grid"
    style="display: grid; grid-template-columns: repeat(8, 32px); gap: 8px; padding: 8px; background: var(--surface-default); color: var(--text-primary);"
>
    {#each ICON_NAMES as name (name)}
        <div style="display: flex; align-items: center; justify-content: center; width: 32px; height: 32px;">
            <Icon {name} size={20} />
        </div>
    {/each}
</div>
