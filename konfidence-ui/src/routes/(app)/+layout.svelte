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
    import "@ui5/webcomponents-fiori/dist/UserMenu.js";
    import "@ui5/webcomponents-fiori/dist/UserMenuAccount.js";
    import "@ui5/webcomponents-fiori/dist/UserMenuItem.js";
    import "@ui5/webcomponents-icons/dist/action-settings.js";
    import "@ui5/webcomponents-icons/dist/database.js";
    import "@ui5/webcomponents-icons/dist/grid.js";
    import "@ui5/webcomponents-icons/dist/menu2.js";
    import "@ui5/webcomponents-icons/dist/palette.js";
    import "@ui5/webcomponents-icons/dist/radar-chart.js";
    import "@ui5/webcomponents-icons/dist/upstacked-chart.js";

    import {
        DEFAULT_SETTINGS_TAB,
        SETTINGS_URL_PARAM,
        type SettingsTab,
        parseSettingsTab,
    } from "$lib/components/settings/settings-tab.js";
    import {
        StageCardVariantPreference,
        setStageCardVariantPreference,
    } from "$lib/stores/stage-card-variant.svelte";
    import ProjectSelector from "$lib/components/ProjectSelector.svelte";
    import { sidebar, toggleSidebar } from "$lib/stores/sidebar.svelte";
    import { themePreference } from "$lib/stores/theme.svelte";
    import type { LayoutProps } from "./$types";
    import type { ResolvedPathname } from "$app/types";
    import SettingsDialog from "$lib/components/settings/SettingsDialog.svelte";
    import { SvelteURLSearchParams } from "svelte/reactivity";
    import { afterNavigate, goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { tick } from "svelte";

    type SideNavigationItemClickEventDetail =
        import("@ui5/webcomponents-fiori/dist/SideNavigationItemBase.js").SideNavigationItemClickEventDetail;
    type ShellBarProfileClickEventDetail =
        import("@ui5/webcomponents-fiori/dist/ShellBar.js").ShellBarProfileClickEventDetail;
    type UserMenuItemClickEventDetail =
        import("@ui5/webcomponents-fiori/dist/UserMenu.js").UserMenuItemClickEventDetail;

    type NavHref =
        | `/projects/${string}/landscape`
        | `/projects/${string}/vector-deployments`
        | `/projects/${string}/artifact-deployments`;

    interface NavItem {
        href: NavHref;
        icon: string;
        text: string;
    }
    interface NavGroup {
        items: readonly NavItem[];
        text: string;
    }

    const SETTINGS_MENU_ITEM_ID = "settings";

    const DARK_LOGO_SRC = "/assets/logo/full/SVG/400_konfidence_logo_dark.svg";
    const LIGHT_LOGO_SRC = "/assets/logo/full/SVG/400_konfidence_logo_light.svg";
    const AVATAR_INITIALS_MAX = 2;

    const { children, data }: LayoutProps = $props();

    const stageCardVariantPreference = new StageCardVariantPreference();
    setStageCardVariantPreference(stageCardVariantPreference);

    let userMenuOpen = $state(false);
    let userMenuOpener = $state<HTMLElement | undefined>();
    let selectedProjectId = $state(page.data.project?.id);

    afterNavigate(() => {
        if (page.data.project) {
            selectedProjectId = page.data.project.id;
        }
    });

    const settingsTab = $derived<SettingsTab | undefined>(
        parseSettingsTab(page.url.searchParams.get(SETTINGS_URL_PARAM) ?? undefined),
    );
    const settingsOpen = $derived(settingsTab !== undefined);

    const buildSettingsUrl = (tab: SettingsTab | undefined): ResolvedPathname => {
        const params = new SvelteURLSearchParams(page.url.searchParams);
        if (tab === undefined) {
            params.delete(SETTINGS_URL_PARAM);
        } else {
            params.set(SETTINGS_URL_PARAM, tab);
        }
        const query = params.toString();
        if (query.length === 0) {
            return page.url.pathname;
        }
        return `${page.url.pathname}?${query}` as ResolvedPathname;
    };

    const openSettings = (tab: SettingsTab = DEFAULT_SETTINGS_TAB): void => {
        const url: ResolvedPathname = buildSettingsUrl(tab);
        void goto(url, { keepFocus: true, noScroll: true });
    };

    const changeSettingsTab = (tab: SettingsTab): void => {
        const url: ResolvedPathname = buildSettingsUrl(tab);
        void goto(url, {
            keepFocus: true,
            noScroll: true,
            replaceState: true,
        });
    };

    const closeSettings = (): void => {
        if (!settingsOpen) {
            return;
        }
        const url: ResolvedPathname = buildSettingsUrl(undefined);
        void goto(url, {
            keepFocus: true,
            noScroll: true,
            replaceState: true,
        });
    };

    const logoSrc = $derived.by(() => {
        if (themePreference.selected.includes("dark")) {
            return DARK_LOGO_SRC;
        }
        return LIGHT_LOGO_SRC;
    });

    const accountSubtitle = $derived(data.user.email);
    const selectedProject = $derived(
        data.projects.find((project) => project.id === selectedProjectId),
    );
    const navGroups = $derived.by((): readonly NavGroup[] => {
        const groups: NavGroup[] = [];
        if (selectedProject) {
            const projectId = selectedProject.id;
            groups.push({
                items: [
                    {
                        href: `/projects/${projectId}/landscape`,
                        icon: "upstacked-chart",
                        text: "Landscape",
                    },
                    {
                        href: `/projects/${projectId}/vector-deployments`,
                        icon: "radar-chart",
                        text: "Vector Deployments",
                    },
                    {
                        href: `/projects/${projectId}/artifact-deployments`,
                        icon: "database",
                        text: "Artifact Deployments",
                    },
                ],
                text: "Delivery",
            });
        }
        return groups;
    });

    const avatarInitials = $derived(
        data.user.name
            .split(" ")
            .map((part: string) => part[0])
            .join("")
            .slice(0, AVATAR_INITIALS_MAX)
            .toUpperCase(),
    );

    const handleProfileClick = (event: CustomEvent<ShellBarProfileClickEventDetail>): void => {
        userMenuOpener = event.detail.targetRef;
        userMenuOpen = true;
    };

    const handleUserMenuClose = (): void => {
        userMenuOpen = false;
    };

    const handleUserMenuItemClick = (
        event: CustomEvent<UserMenuItemClickEventDetail>,
    ): void => {
        if (event.detail.item.getAttribute("data-id") !== SETTINGS_MENU_ITEM_ID) {
            return;
        }
        userMenuOpen = false;
        openSettings(DEFAULT_SETTINGS_TAB);
    };

    const handleSignOutClick = async (): Promise<void> => {
        userMenuOpen = false;
        const response = await globalThis.fetch("/api/logout", { method: "POST" });
        if (response.ok) {
            globalThis.location.assign("/");
        }
    };

    const selectProject = (projectId: string): void => {
        selectedProjectId = projectId;
        goto(resolve(`/projects/${projectId}/landscape`));
    };

    const handleSideNavClick = (
        event: CustomEvent<SideNavigationItemClickEventDetail>,
        href: NavItem["href"],
    ): void => {
        event.preventDefault();
        // eslint-disable-next-line svelte/no-navigation-without-resolve -- These hrefs are already resolved project paths.
        goto(href);
    };
</script>

<ui5-navigation-layout id="nl1" mode={sidebar.mode}>
    <ui5-shellbar
        slot="header"
        notifications-count="3"
        show-notifications
        onui5-profile-click={handleProfileClick}
    >
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
                src={logoSrc}
                alt="Konfidence"
            />
        </ui5-shellbar-branding>
        <ui5-tag design="Set2" color-scheme="5" slot="content">UI5 Web Components</ui5-tag>
        <ui5-avatar
            slot="profile"
            initials={avatarInitials}
            color-scheme="Accent6"
            accessible-name={`Open account menu for ${data.user.name}`}
        ></ui5-avatar>
    </ui5-shellbar>
    <ui5-side-navigation class="side-panel" slot="sideContent">
     {#if sidebar.mode !== "Collapsed"}
        <ProjectSelector
            projects={data.projects}
            selectedProjectId={selectedProject?.id}
            onselect={selectProject}
        />
        {:else}
        <ui5-side-navigation-item
            text="Select a project"
            icon="grid"
            selected={false}
            onui5-click={() => {
                sidebar.mode = "Expanded";
                tick().then(() => {
                    const select = document.querySelector("ui5-select#project-select") as any;
                    if (select) {
                        select._onclick();
                    }
                });
            }}
        ></ui5-side-navigation-item>
     {/if}
        {#each navGroups as navGroup (navGroup.text)}
                <ui5-side-navigation-group text={navGroup.text} expanded>
                    {#each navGroup.items as navItem (navItem.href)}
                        <ui5-side-navigation-item
                            text={navItem.text}
                            href={navItem.href}
                            icon={navItem.icon}
                            selected={page.url.pathname.startsWith(navItem.href)}
                            onui5-click={(
                                event: CustomEvent<SideNavigationItemClickEventDetail>,
                            ) => handleSideNavClick(event, navItem.href)}
                        ></ui5-side-navigation-item>
                    {/each}
                </ui5-side-navigation-group>
            {/each}
    </ui5-side-navigation>
    <div class="content">
        {@render children()}
    </div>
</ui5-navigation-layout>

<ui5-user-menu
    id="user-menu"
    open={userMenuOpen}
    opener={userMenuOpener}
    onui5-close={handleUserMenuClose}
    onui5-item-click={handleUserMenuItemClick}
    onui5-sign-out-click={handleSignOutClick}
>
    <ui5-user-menu-account
        slot="accounts"
        title-text={data.user.name}
        subtitle-text={accountSubtitle}
        selected
    ></ui5-user-menu-account>

    <ui5-user-menu-item
        icon="action-settings"
        text="Settings"
        data-id={SETTINGS_MENU_ITEM_ID}
    ></ui5-user-menu-item>
</ui5-user-menu>

<SettingsDialog
    open={settingsOpen}
    tab={settingsTab ?? DEFAULT_SETTINGS_TAB}
    user={data.user}
    onClose={closeSettings}
    onTabChange={changeSettingsTab}
/>

<style>
    :global(ui5-navigation-layout) {
        height: 100%;
    }

    :global(ui5-shellbar) {
        border-bottom: 1px solid var(--sapList_BorderColor);
        box-shadow: var(--sapContent_Shadow0);
    }

    :global(ui5-side-navigation) {
        flex: 1;
        border-right: 1px solid var(--sapList_BorderColor);
    }

    .side-panel {
        display: flex;
        flex-direction: column;
        height: 100%;
        background: var(--sapList_Background);
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
