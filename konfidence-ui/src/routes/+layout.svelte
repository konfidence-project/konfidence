<script lang="ts">
    import "../theme/app.css";
    import { ThemePreference, setThemePreference } from "$lib/theme-preference.svelte";
    import type { LayoutProps } from "./$types";
    import { untrack } from "svelte";

    const { children, data }: LayoutProps = $props();
    const theme = new ThemePreference(untrack(() => data.theme));
    setThemePreference(theme);
</script>

<svelte:head>
    <title>Konfidence</title>
    <link
        rel="icon"
        href="/assets/logo/Icon_only/SVG/32_konfidence_icon_color.svg"
    />
    <meta name="theme-color" content={theme.selected === "konfidence-dark" ? "#111318" : "#ffffff"} />
</svelte:head>

{@render children()}
