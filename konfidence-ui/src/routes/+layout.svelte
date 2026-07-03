<script lang="ts">
    import "@ui5/webcomponents/dist/Avatar.js";
    import "@ui5/webcomponents/dist/Button.js";
    import "@ui5/webcomponents/dist/ToggleButton.js";

    import "@ui5/webcomponents-fiori/dist/NavigationLayout.js";
    import "@ui5/webcomponents-fiori/dist/ShellBar.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarBranding.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarItem.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarSearch.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigation.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigationGroup.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigationItem.js";

    import "@ui5/webcomponents-icons/dist/AllIcons.js";

    import {
        StageCardVariantPreference,
        setStageCardVariantPreference,
    } from "$lib/stores/stage-card-variant.svelte";
    import {
        initTheme,
        markCustomThemeStylesheetLoaded,
        themePreference,
    } from "$lib/stores/theme.svelte";
    import { sidebar, toggleSidebar } from "$lib/stores/sidebar.svelte";
    import { goto } from "$app/navigation";
    import konfidenceDarkThemeHref from "../theme/konfidence-dark.css?url";
    import konfidenceThemeHref from "../theme/konfidence.css?url";
    import { page } from "$app/state";
    import { resolve } from "$app/paths";

    type SideNavigationItemClickEventDetail = import("@ui5/webcomponents-fiori/dist/SideNavigationItemBase.js").SideNavigationItemClickEventDetail;
    type UITheme = import("$lib/stores/theme.svelte").UITheme;

    const { children } = $props();

    const customThemeLinks = [
        { href: konfidenceThemeHref, id: "konfidence" },
        { href: konfidenceDarkThemeHref, id: "konfidence-dark" },
    ] as const satisfies readonly { id: UITheme; href: string }[];

    interface NavGroup {
        text: string;
        items: readonly {
            text: string;
            href: "/" | "/landscape" | "/vectors" | "/promotions" | "/settings";
            icon: string;
        }[];
    }

    const navGroups: readonly NavGroup[] = [
        {
            items: [
                {
                    href: "/landscape",
                    icon: "upstacked-chart",
                    text: "Landscape",
                },
                {
                    href: "/vectors",
                    icon: "radar-chart",
                    text: "Vectors",
                },
                {
                    href: "/promotions",
                    icon: "process",
                    text: "Promotions",
                },
            ],
            text: "Delivery",
        },
        {
            items: [
                {
                    href: "/settings",
                    icon: "action-settings",
                    text: "Settings",
                },
            ],
            text: "Administration",
        },
    ];

    initTheme();
    setStageCardVariantPreference(new StageCardVariantPreference());
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

<ui5-navigation-layout id="nl1" mode={sidebar.mode}>
    <ui5-shellbar slot="header" notifications-count="3" show-notifications>
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-button is an interactive web component with built-in keyboard handling) -->
        <ui5-button
            icon="menu2"
            slot="startButton"
            id="startButton"
            onclick={toggleSidebar}
        ></ui5-button>
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-shellbar-branding is an interactive web component with built-in keyboard handling) -->
        <ui5-shellbar-branding
            slot="branding"
            class="brand"
            accessible-name="Konfidence home"
            onclick={() => goto(resolve("/"))}
        >
            <img
                slot="logo"
                class="brand-logo"
                src={themePreference.selected === "konfidence-dark"
                    ? "/assets/logo/full/SVG/400_konfidence_logo_dark.svg"
                    : "/assets/logo/full/SVG/400_konfidence_logo_light.svg"}
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
        {#each navGroups as navGroup (navGroup.text)}
            <ui5-side-navigation-group text={navGroup.text} expanded>
                {#each navGroup.items as navItem (navItem.href)}
                    <ui5-side-navigation-item
                        text={navItem.text}
                        href={navItem.href}
                        icon={navItem.icon}
                        selected={page.url.pathname === navItem.href}
                        onui5-click={(
                            event: CustomEvent<SideNavigationItemClickEventDetail>,
                        ) => {
                            event.preventDefault();
                            goto(resolve(navItem.href));
                        }}
                    ></ui5-side-navigation-item>
                {/each}
            </ui5-side-navigation-group>
        {/each}
    </ui5-side-navigation>
    <div class="content">
        {@render children()}
    </div>
</ui5-navigation-layout>

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

    :global(ui5-navigation-layout) {
        height: 100%;
    }

    :global(ui5-shellbar) {
        border-bottom: 1px solid var(--sapList_BorderColor);
        box-shadow: var(--sapContent_Shadow0);
    }

    :global(ui5-side-navigation) {
        border-right: 1px solid var(--sapList_BorderColor);
    }

    .brand-logo {
        width: 172px;
        height: 20px;
        object-fit: contain;
        display: block;
    }

    .content {
        display: flex;
        flex-direction: column;
        padding: 0;
        margin: 0;
        height: 100%;
        box-sizing: border-box;
    }
</style>
