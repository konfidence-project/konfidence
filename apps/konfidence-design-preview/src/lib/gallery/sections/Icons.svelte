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
    <p class="count mb-3 text-meta text-content-secondary">{ICON_NAMES.length} icons in the curated subset</p>
    <div class="grid grid-cols-[repeat(auto-fill,minmax(120px,1fr))] gap-3">
        {#each ICON_NAMES as name (name)}
            <div class="cell flex flex-col items-center gap-1.5 rounded-lg border border-outline-subtle bg-surface-default px-2 py-2.5 [&_code]:text-center [&_code]:font-mono [&_code]:text-xs [&_code]:text-content-tertiary [&_ui5-icon.icon]:size-6 [&_ui5-icon.icon]:text-content-primary">
                <ui5-icon class="icon" {name}></ui5-icon>
                <code>{name}</code>
            </div>
        {/each}
    </div>
</Section>
