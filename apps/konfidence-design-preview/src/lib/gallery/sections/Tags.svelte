<script lang="ts">
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";
    import { TAG_COLORS } from "$lib/tokens/read-tokens";
</script>

<Section
    id="foundation-tags"
    title="Foundation · Category tags"
    subtitle="Twelve accessible tag palettes. Each color also carries a distinct icon (double-coding) in the reference design."
    status="placeholder"
>
    <div class="preview mb-4 flex flex-wrap gap-2">
        {#each TAG_COLORS as color (color)}
            <span
                class="tag inline-flex items-center rounded-full border border-transparent px-2.5 py-[3px] text-meta font-ui-semibold capitalize"
                style:background="var(--tag-{color}-bg, var(--surface-subtle))"
                style:color="var(--tag-{color}-fg, var(--text-secondary))"
                style:border-color="var(--tag-{color}-bd, var(--border-subtle))"
            >
                {color}
            </span>
        {/each}
    </div>

    <NotYetImplemented
        note="The tag colors are defined as CSS custom properties in the design system, but no Svelte `Tag` component exists yet. The preview above uses raw CSS custom properties."
        planned={["<Tag color size icon>", "<MetaTag>", "Tag double-coding (icon + color)"]}
    />
</Section>
