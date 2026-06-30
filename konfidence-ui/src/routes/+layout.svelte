<script lang="ts">
    import "../theme/konfidence.css";
    import { setTheme } from "@ui5/webcomponents-base/dist/config/Theme.js";
    import "@ui5/webcomponents/dist/Button.js";
    import "@ui5/webcomponents/dist/ToggleButton.js";
    import "@ui5/webcomponents/dist/Avatar.js";

    import "@ui5/webcomponents-fiori/dist/NavigationLayout.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigation.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigationGroup.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigationItem.js";
    import "@ui5/webcomponents-fiori/dist/ShellBar.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarBranding.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarSearch.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarItem.js";

    import "@ui5/webcomponents-icons/dist/upstacked-chart.js";
    import "@ui5/webcomponents-icons/dist/menu2.js";
    import "@ui5/webcomponents-icons/dist/sys-help.js";
    import "@ui5/webcomponents-icons/dist/da.js";

    setTheme("konfidence");

    let { children } = $props();

    import { sidebar, toggleSidebar } from "$lib/stores/sidebar.svelte";
</script>

<svelte:head>
    <link
        rel="icon"
        href="/assets/logo/Icon_only/SVG/32_konfidence_icon_color.svg"
    />
</svelte:head>

<ui5-navigation-layout id="nl1" mode="{sidebar.mode}">
    <ui5-shellbar slot="header" notifications-count="3" show-notifications>
        <ui5-button icon="menu2" slot="startButton" id="startButton" onclick={toggleSidebar}
        ></ui5-button>
        <ui5-shellbar-branding slot="branding" class="brand">
            <img
                slot="logo"
                class="brand-logo"
                src="/assets/logo/full/SVG/400_konfidence_logo_light.svg"
                alt="Konfidence"
            />
        </ui5-shellbar-branding>
        <ui5-avatar slot="profile">
            <img
                src="https://ui5.github.io/webcomponents/images/avatars/man_avatar_3.png"
                alt="User profile"
            />
        </ui5-avatar>
    </ui5-shellbar>
    <ui5-side-navigation id="sn1" slot="sideContent">
        <ui5-side-navigation-group text="Stages" expanded>
            <ui5-side-navigation-item
                text="All Stages"
                href="/"
                icon="upstacked-chart"
            ></ui5-side-navigation-item>
        </ui5-side-navigation-group>
    </ui5-side-navigation>
    <div class="content">
        {@render children()}
    </div>
</ui5-navigation-layout>

<style>
    :global(:root) {
        --sapFontFamily:
            "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }

    :global(html),
    :global(body) {
        min-height: 100%;
        margin: 0;
        font-family:
            "Inter",
            -apple-system,
            BlinkMacSystemFont,
            "Segoe UI",
            sans-serif;
        background:
            radial-gradient(
                circle at top left,
                rgba(128, 210, 224, 0.18),
                transparent 34rem
            ),
            linear-gradient(180deg, #f7fbfc 0%, #eef7f9 100%);
        color: #102a38;
    }

    :global(ui5-shellbar) {
        border-bottom: 1px solid rgba(19, 156, 199, 0.16);
        box-shadow: 0 18px 44px rgba(14, 127, 163, 0.08);
    }

    :global(ui5-side-navigation) {
        border-right: 1px solid rgba(19, 156, 199, 0.14);
    }

    .brand-logo {
        width: 172px;
        height: 20px;
        object-fit: contain;
        display: block;
    }

    .content {
        min-height: calc(100vh - 4rem);
        padding: 2.5rem clamp(1rem, 3vw, 3rem);
        box-sizing: border-box;
    }
</style>
