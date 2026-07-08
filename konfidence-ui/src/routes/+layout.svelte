<script lang="ts">
    import {
        initTheme,
        markCustomThemeStylesheetLoaded,
        themePreference,
    } from "$lib/stores/theme.svelte";
    import konfidenceDarkThemeHref from "../theme/konfidence-dark.css?url";
    import konfidenceThemeHref from "../theme/konfidence.css?url";

    type UITheme = import("$lib/stores/theme.svelte").UITheme;

    const { children } = $props();

    const customThemeLinks = [
        { href: konfidenceThemeHref, id: "konfidence" },
        { href: konfidenceDarkThemeHref, id: "konfidence-dark" },
    ] as const satisfies readonly { id: UITheme; href: string }[];

    initTheme();
</script>

<svelte:head>
    <link
        rel="icon"
        href="/assets/logo/Icon_only/SVG/32_konfidence_icon_color.svg"
    />
    {#each customThemeLinks as customTheme (customTheme.id)}
        <link
            rel="stylesheet"
            href={customTheme.href}
            media={themePreference.selected === customTheme.id
                ? "all"
                : "not all"}
            data-konfidence-theme={customTheme.id}
            onload={() => markCustomThemeStylesheetLoaded(customTheme.id)}
        />
    {/each}
</svelte:head>

{@render children()}

<style>
    :global(:root) {
        --konfidence-sidebar-status-color: var(--sapPositiveElementColor);
    }

    :global(html),
    :global(body) {
        height: 100vh;
        margin: 0;
        padding: 0;
        font-family: var(--sapFontFamily), sans-serif;
        background:
            radial-gradient(
                circle at top left,
                color-mix(in srgb, var(--sapHighlightColor) 14%, transparent),
                transparent 34rem
            ),
            var(--sapBackgroundColor);
        color: var(--sapTextColor);
    }
</style>
