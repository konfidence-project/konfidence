<script lang="ts">
    import "../theme/app.css";
    import { setupI18n } from "$lib/i18n";
    import { LocalePreference, setLocalePreference } from "$lib/locale-preference.svelte";
    import { ThemePreference, setThemePreference } from "$lib/theme-preference.svelte";
    import type { LayoutProps } from "./$types";
    import { untrack } from "svelte";

    const { children, data }: LayoutProps = $props();
    setupI18n();
    const locale = new LocalePreference(untrack(() => data.locale));
    const theme = new ThemePreference(untrack(() => data.theme));
    setLocalePreference(locale);
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
