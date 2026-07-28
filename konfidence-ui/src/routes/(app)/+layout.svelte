<script lang="ts">
    import BellIcon from "@lucide/svelte/icons/bell";
    import LogOutIcon from "@lucide/svelte/icons/log-out";
    import MenuIcon from "@lucide/svelte/icons/menu";
    import SettingsIcon from "@lucide/svelte/icons/settings";
    import AppNavigation from "$lib/components/AppNavigation.svelte";
    import SettingsDialog from "$lib/components/settings/SettingsDialog.svelte";
    import {
        DEFAULT_SETTINGS_TAB,
        SETTINGS_URL_PARAM,
        type SettingsTab,
        parseSettingsTab,
    } from "$lib/components/settings/settings-tab.js";
    import * as Avatar from "$lib/components/ui/avatar/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
    import * as Sheet from "$lib/components/ui/sheet/index.js";
    import { getThemePreference } from "$lib/stores/theme.svelte";
    import { afterNavigate, goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import type { ResolvedPathname } from "$app/types";
    import { SvelteURLSearchParams } from "svelte/reactivity";
    import type { LayoutProps } from "./$types";

    const { children, data }: LayoutProps = $props();
    const theme = getThemePreference();
    const AVATAR_INITIALS_MAX = 2;

    let desktopNavigationOpen = $state(true);
    let mobileNavigationOpen = $state(false);
    let selectedProjectId = $state(page.data.project?.id);
    let accountButton = $state<HTMLButtonElement | null>(null);

    afterNavigate(() => {
        selectedProjectId = page.data.project?.id;
    });

    const selectedProject = $derived(
        data.projects.find((project) => project.id === selectedProjectId),
    );
    const settingsTab = $derived(
        parseSettingsTab(page.url.searchParams.get(SETTINGS_URL_PARAM) ?? undefined),
    );
    const settingsOpen = $derived(settingsTab !== undefined);
    const avatarInitials = $derived(
        data.user.name
            .split(" ")
            .map((part: string) => part[0] ?? "")
            .join("")
            .slice(0, AVATAR_INITIALS_MAX)
            .toUpperCase(),
    );
    const logoSrc = $derived.by(() => {
        if (theme.selected === "konfidence-dark") {
            return "/assets/logo/full/SVG/400_konfidence_logo_dark.svg";
        }
        return "/assets/logo/full/SVG/400_konfidence_logo_light.svg";
    });
    const navItems = $derived.by(() => {
        if (!selectedProject) {
            return [];
        }
        return [
            {
                href: `/projects/${selectedProject.id}/landscape`,
                icon: "landscape" as const,
                text: "Landscape",
            },
            {
                href: `/projects/${selectedProject.id}/vector-deployments`,
                icon: "vector" as const,
                text: "Vector Deployments",
            },
        ];
    });

    const buildSettingsUrl = (tab: SettingsTab | undefined): ResolvedPathname => {
        const params = new SvelteURLSearchParams(page.url.searchParams);
        if (tab) {
            params.set(SETTINGS_URL_PARAM, tab);
        } else {
            params.delete(SETTINGS_URL_PARAM);
        }
        const query = params.toString();
        if (query) {
            return `${page.url.pathname}?${query}` as ResolvedPathname;
        }
        return page.url.pathname;
    };

    const openSettings = (): void => {
        void goto(buildSettingsUrl(DEFAULT_SETTINGS_TAB), { keepFocus: true, noScroll: true });
    };

    const changeSettingsTab = (tab: SettingsTab): void => {
        void goto(buildSettingsUrl(tab), { keepFocus: true, noScroll: true, replaceState: true });
    };

    const closeSettings = (): void => {
        void goto(buildSettingsUrl(undefined), {
            keepFocus: true,
            noScroll: true,
            replaceState: true,
        });
    };

    const toggleNavigation = (): void => {
        if (globalThis.matchMedia("(max-width: 48rem)").matches) {
            mobileNavigationOpen = true;
        } else {
            desktopNavigationOpen = !desktopNavigationOpen;
        }
    };

    const selectProject = (projectId: string): void => {
        selectedProjectId = projectId;
        mobileNavigationOpen = false;
        void goto(resolve(`/projects/${projectId}/landscape`));
    };

    const signOut = async (): Promise<void> => {
        const response = await globalThis.fetch("/api/logout", { method: "POST" });
        if (response.ok) {
            globalThis.location.assign("/");
        }
    };
</script>

<div
    class={[
        "grid min-h-dvh grid-rows-[3.25rem_minmax(0,1fr)] transition-[grid-template-columns] duration-[160ms] motion-reduce:transition-none max-[48rem]:flex max-[48rem]:flex-col",
        desktopNavigationOpen
            ? "grid-cols-[16rem_minmax(0,1fr)]"
            : "grid-cols-[0_minmax(0,1fr)]",
    ]}
>
    <header
        class="z-20 col-span-full flex items-center gap-3 border-b bg-card/94 px-4 shadow-[0_1px_4px_color-mix(in_oklch,var(--foreground)_10%,transparent)] backdrop-blur-[0.75rem] max-[48rem]:sticky max-[48rem]:top-0 max-[48rem]:min-h-[3.25rem]"
    >
        <Button
            variant="ghost"
            size="icon"
            aria-label="Toggle navigation"
            onclick={toggleNavigation}
        >
            <MenuIcon />
        </Button>
        <a class="inline-flex items-center" href={resolve("/")} aria-label="Konfidence home">
            <img
                class="block h-[1.3rem] w-[10.5rem] object-contain max-[48rem]:w-[9.25rem]"
                src={logoSrc}
                alt="Konfidence"
            />
        </a>
        <div class="ml-auto flex items-center gap-1">
            <Button class="relative" variant="ghost" size="icon" aria-label="Notifications, 3 unread">
                <BellIcon />
                <span
                    class="absolute -top-[0.15rem] -right-[0.05rem] grid h-4 min-w-4 place-items-center rounded-full bg-destructive text-[0.65rem] font-bold text-white"
                    aria-hidden="true">3</span
                >
            </Button>
            <DropdownMenu.Root>
                <DropdownMenu.Trigger>
                    {#snippet child({ props })}
                        <Button
                            {...props}
                            bind:ref={accountButton}
                            variant="ghost"
                            size="icon"
                            aria-label={`Open account menu for ${data.user.name}`}
                        >
                            <Avatar.Root class="size-8">
                                <Avatar.Fallback>{avatarInitials}</Avatar.Fallback>
                            </Avatar.Root>
                        </Button>
                    {/snippet}
                </DropdownMenu.Trigger>
                <DropdownMenu.Content align="end" class="w-64">
                    <DropdownMenu.Label>
                        <span class="block truncate">{data.user.name}</span>
                        <span class="block truncate text-xs font-normal text-muted-foreground">
                            {data.user.email}
                        </span>
                    </DropdownMenu.Label>
                    <DropdownMenu.Separator />
                    <DropdownMenu.Item onSelect={openSettings}><SettingsIcon /> Settings</DropdownMenu.Item>
                    <DropdownMenu.Item variant="destructive" onSelect={signOut}>
                        <LogOutIcon /> Sign Out
                    </DropdownMenu.Item>
                </DropdownMenu.Content>
            </DropdownMenu.Root>
        </div>
    </header>

    <aside
        class="min-w-0 overflow-x-hidden overflow-y-auto border-r border-sidebar-border bg-sidebar text-sidebar-foreground max-[48rem]:hidden"
        aria-label="Project navigation"
    >
        <AppNavigation
            projects={data.projects}
            selectedProjectId={selectedProject?.id}
            items={navItems}
            currentPath={page.url.pathname}
            onselect={selectProject}
            selectorId="project-select-desktop"
        />
    </aside>

    <main
        class="flex min-h-0 min-w-0 flex-col overflow-auto max-[48rem]:min-h-[calc(100dvh-3.25rem)] max-[48rem]:flex-1 max-[48rem]:overflow-visible"
        id="main-content"
    >
        {@render children()}
    </main>
</div>

<Sheet.Root bind:open={mobileNavigationOpen}>
    <Sheet.Content
        side="left"
        class="mobile-navigation !w-[88vw] !max-w-[20rem] p-0"
        aria-label="Project navigation"
    >
        <Sheet.Header class="sr-only">
            <Sheet.Title>Project navigation</Sheet.Title>
            <Sheet.Description>Select a project route.</Sheet.Description>
        </Sheet.Header>
        <AppNavigation
            projects={data.projects}
            selectedProjectId={selectedProject?.id}
            items={navItems}
            currentPath={page.url.pathname}
            onselect={selectProject}
            onNavigate={() => (mobileNavigationOpen = false)}
            projectSelectorClass="pr-15"
            selectorId="project-select-mobile"
        />
    </Sheet.Content>
</Sheet.Root>

<SettingsDialog
    open={settingsOpen}
    tab={settingsTab ?? DEFAULT_SETTINGS_TAB}
    user={data.user}
    onClose={closeSettings}
    onTabChange={changeSettingsTab}
    returnFocus={() => accountButton?.focus()}
/>
