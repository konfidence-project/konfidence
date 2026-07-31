<script lang="ts">
    import "../app.css";
    import { ThemePreference, setThemePreference } from "$lib/stores/theme.svelte";
    import type { LayoutProps } from "./$types";

    const { children, data }: LayoutProps = $props();
    const theme = new ThemePreference(() => data.theme);
    setThemePreference(theme);
</script>

<svelte:head>
    <title>Konfidence</title>
    <link
        rel="icon"
        href="/assets/logo/Icon_only/SVG/32_konfidence_icon_color.svg"
    />
    <meta name="theme-color" content={theme.resolved === "konfidence-dark" ? "#111111" : "#ffffff"} />
</svelte:head>

{@render children()}
