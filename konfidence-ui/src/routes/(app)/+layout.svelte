<script lang="ts">
    import "@ui5/webcomponents/dist/Avatar.js";
    import "@ui5/webcomponents/dist/Button.js";
    import "@ui5/webcomponents/dist/Menu.js";
    import "@ui5/webcomponents/dist/MenuItem.js";
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
    import "@ui5/webcomponents-icons/dist/dark-mode.js";
    import "@ui5/webcomponents-icons/dist/database.js";
    import "@ui5/webcomponents-icons/dist/grid.js";
    import "@ui5/webcomponents-icons/dist/light-mode.js";
    import "@ui5/webcomponents-icons/dist/menu2.js";
    import "@ui5/webcomponents-icons/dist/palette.js";
    import "@ui5/webcomponents-icons/dist/radar-chart.js";
    import "@ui5/webcomponents-icons/dist/sys-monitor.js";
    import "@ui5/webcomponents-icons/dist/upstacked-chart.js";
    import "@ui5/webcomponents-icons/dist/world.js";

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
    import {
        SUPPORTED_LANGUAGES,
        languagePreference,
        selectLanguage,
        t,
    } from "$lib/stores/i18n.svelte";
    import {
        selectThemeMode,
        themeModePreference,
        themePreference,
    } from "$lib/stores/theme.svelte";
    import ProjectSelector from "$lib/components/ProjectSelector.svelte";
    import { sidebar, toggleSidebar } from "$lib/stores/sidebar.svelte";
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
    type MenuItemClickEventDetail =
        import("@ui5/webcomponents/dist/Menu.js").MenuItemClickEventDetail;

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
    let themeMenuOpen = $state(false);
    let languageMenuOpen = $state(false);
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

    const themeSwitcherIcon = $derived.by(() => {
        const mode = themeModePreference.selected;
        if (mode === "light") {
            return "light-mode";
        }
        if (mode === "dark") {
            return "dark-mode";
        }
        if (mode === "system") {
            return "sys-monitor";
        }
        // custom
        return "palette";
    });

    const accountSubtitle = $derived(data.user.email);
    const selectedProject = $derived(
        data.projects.find((project) => project.id === selectedProjectId),
    );
    const navGroups = $derived.by((): readonly NavGroup[] => {
        // Read bundle version so nav labels update on language change.
        const { bundleVersion } = languagePreference;
        void bundleVersion;
        const groups: NavGroup[] = [];
        if (selectedProject) {
            const projectId = selectedProject.id;
            groups.push({
                items: [
                    {
                        href: `/projects/${projectId}/landscape`,
                        icon: "upstacked-chart",
                        text: t("APP_NAV_LANDSCAPE"),
                    },
                    {
                        href: `/projects/${projectId}/vector-deployments`,
                        icon: "radar-chart",
                        text: t("APP_NAV_VECTOR_DEPLOYMENTS"),
                    },
                    {
                        href: `/projects/${projectId}/artifact-deployments`,
                        icon: "database",
                        text: t("APP_NAV_ARTIFACT_DEPLOYMENTS"),
                    },
                ],
                text: t("APP_NAV_GROUP_DELIVERY"),
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

    const handleThemeItemClick = (): void => {
        languageMenuOpen = false;
        themeMenuOpen = true;
    };

    const handleThemeMenuClose = (): void => {
        themeMenuOpen = false;
    };

    const handleThemeMenuItemClick = (event: CustomEvent<MenuItemClickEventDetail>): void => {
        const mode = event.detail.item.getAttribute("data-id");
        if (mode) {
            selectThemeMode(mode);
        }
        themeMenuOpen = false;
    };

    const handleLanguageItemClick = (): void => {
        themeMenuOpen = false;
        languageMenuOpen = true;
    };

    const handleLanguageMenuClose = (): void => {
        languageMenuOpen = false;
    };

    const handleLanguageMenuItemClick = (event: CustomEvent<MenuItemClickEventDetail>): void => {
        const mode = event.detail.item.getAttribute("data-id");
        if (mode) {
            selectLanguage(mode);
        }
        languageMenuOpen = false;
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
            accessible-name={t("APP_BRANDING_HOME")}
            onclick={() => goto(resolve("/"))}
        >
            <img
                slot="logo"
                class="brand-logo"
                src={logoSrc}
                alt={t("APP_BRANDING_ALT")}
            />
        </ui5-shellbar-branding>
        <ui5-tag design="Set2" color-scheme="5" slot="content">{t("APP_TAG_UI5")}</ui5-tag>
        <ui5-shellbar-item
            id="theme-switcher"
            icon={themeSwitcherIcon}
            text={t("SHELLBAR_THEME_TEXT")}
            tooltip={t("SHELLBAR_THEME_TOOLTIP")}
            onui5-click={handleThemeItemClick}
        ></ui5-shellbar-item>
        <ui5-shellbar-item
            id="language-switcher"
            icon="world"
            text={t("SHELLBAR_LANGUAGE_TEXT")}
            tooltip={t("SHELLBAR_LANGUAGE_TOOLTIP")}
            onui5-click={handleLanguageItemClick}
        ></ui5-shellbar-item>
        <ui5-avatar
            slot="profile"
            initials={avatarInitials}
            color-scheme="Accent6"
            accessible-name={t("APP_AVATAR_OPEN_ACCOUNT_MENU", data.user.name)}
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
            text={t("APP_SIDENAV_SELECT_PROJECT")}
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

<ui5-menu
    id="theme-menu"
    open={themeMenuOpen}
    opener="theme-switcher"
    onui5-close={handleThemeMenuClose}
    onui5-item-click={handleThemeMenuItemClick}
>
    <ui5-menu-item
        icon="light-mode"
        text={t("THEME_MODE_LIGHT")}
        data-id="light"
        additional-text={themeModePreference.selected === "light" ? "\u2713" : ""}
    ></ui5-menu-item>
    <ui5-menu-item
        icon="dark-mode"
        text={t("THEME_MODE_DARK")}
        data-id="dark"
        additional-text={themeModePreference.selected === "dark" ? "\u2713" : ""}
    ></ui5-menu-item>
    <ui5-menu-item
        icon="sys-monitor"
        text={t("THEME_MODE_SYSTEM")}
        data-id="system"
        additional-text={themeModePreference.selected === "system" ? "\u2713" : ""}
    ></ui5-menu-item>
</ui5-menu>

<ui5-menu
    id="language-menu"
    open={languageMenuOpen}
    opener="language-switcher"
    onui5-close={handleLanguageMenuClose}
    onui5-item-click={handleLanguageMenuItemClick}
>
    <ui5-menu-item
        icon="sys-monitor"
        text={t("LANG_MODE_SYSTEM")}
        data-id="system"
        additional-text={languagePreference.mode === "system" ? "\u2713" : ""}
    ></ui5-menu-item>
    {#each SUPPORTED_LANGUAGES as language (language.id)}
        <ui5-menu-item
            text={t(language.label)}
            data-id={language.id}
            additional-text={languagePreference.mode === language.id ? "\u2713" : ""}
        ></ui5-menu-item>
    {/each}
</ui5-menu>

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
        text={t("APP_MENU_SETTINGS")}
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
