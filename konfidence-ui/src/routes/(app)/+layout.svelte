<script lang="ts">
    import BellIcon from "@lucide/svelte/icons/bell";
    import BoxesIcon from "@lucide/svelte/icons/boxes";
    import LogOutIcon from "@lucide/svelte/icons/log-out";
    import MenuIcon from "@lucide/svelte/icons/menu";
    import NetworkIcon from "@lucide/svelte/icons/network";
    import SettingsIcon from "@lucide/svelte/icons/settings";
    import XIcon from "@lucide/svelte/icons/x";
    import { Avatar, Menu, Portal } from "@skeletonlabs/skeleton-svelte";

    import { afterNavigate, goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import type { ResolvedPathname } from "$app/types";
    import ProjectSelector from "$lib/components/ProjectSelector.svelte";
    import SettingsDialog from "$lib/components/settings/SettingsDialog.svelte";
    import {
        DEFAULT_SETTINGS_TAB,
        SETTINGS_URL_PARAM,
        type SettingsTab,
        parseSettingsTab,
    } from "$lib/components/settings/settings-tab.js";
    import { getThemePreference } from "$lib/theme-preference.svelte";
    import type { LayoutProps } from "./$types";
    import { SvelteURLSearchParams } from "svelte/reactivity";

    const AVATAR_INITIALS_MAX = 2;
    const { children, data }: LayoutProps = $props();
    const theme = getThemePreference();

    let navigationToggled = $state(false);
    let overlayTarget = $state<HTMLElement>();
    let selectedProjectId = $state(page.data.project?.id);

    afterNavigate(() => {
        selectedProjectId = page.data.project?.id;
        navigationToggled = false;
    });

    const selectedProject = $derived(
        data.projects.find((project) => project.id === selectedProjectId),
    );
    const settingsTab = $derived(
        parseSettingsTab(page.url.searchParams.get(SETTINGS_URL_PARAM) ?? undefined),
    );
    const settingsOpen = $derived(settingsTab !== undefined);
    const logoSrc = $derived.by(() => {
        if (theme.selected === "konfidence-dark") {
            return "/assets/logo/full/SVG/400_konfidence_logo_dark.svg";
        }
        return "/assets/logo/full/SVG/400_konfidence_logo_light.svg";
    });
    const avatarInitials = $derived(
        data.user.name
            .split(" ")
            .map((part: string) => part[0] ?? "")
            .join("")
            .slice(0, AVATAR_INITIALS_MAX)
            .toUpperCase(),
    );
    const navigation = $derived.by(() => {
        if (!selectedProject) {
            return [];
        }
        return [
            {
                href: resolve(`/projects/${selectedProject.id}/landscape`),
                icon: NetworkIcon,
                label: "Landscape",
            },
            {
                href: resolve(`/projects/${selectedProject.id}/vector-deployments`),
                icon: BoxesIcon,
                label: "Vector Deployments",
            },
        ];
    });

    const settingsUrl = (tab: SettingsTab | undefined): ResolvedPathname => {
        const parameters = new SvelteURLSearchParams(page.url.searchParams);
        if (tab) {
            parameters.set(SETTINGS_URL_PARAM, tab);
        } else {
            parameters.delete(SETTINGS_URL_PARAM);
        }
        const query = parameters.toString();
        if (query) {
            return `${page.url.pathname}?${query}` as ResolvedPathname;
        }
        return page.url.pathname;
    };

    const openSettings = (): void => {
        void goto(settingsUrl(DEFAULT_SETTINGS_TAB), { keepFocus: true, noScroll: true });
    };

    const selectAccountAction = (value: string): void => {
        if (value === "settings") {
            openSettings();
        } else if (value === "sign-out") {
            void signOut();
        }
    };

    const closeSettings = (): void => {
        void goto(settingsUrl(undefined), {
            keepFocus: true,
            noScroll: true,
            replaceState: true,
        });
    };

    const changeSettingsTab = (tab: SettingsTab): void => {
        void goto(settingsUrl(tab), { keepFocus: true, noScroll: true, replaceState: true });
    };

    const selectProject = (projectId: string): void => {
        selectedProjectId = projectId;
        navigationToggled = false;
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
        "grid min-h-screen grid-cols-[minmax(0,1fr)] grid-rows-[3.35rem_minmax(0,1fr)] bg-app-bg",
        navigationToggled
            ? "min-[52rem]:grid-cols-[4.25rem_minmax(0,1fr)]"
            : "min-[52rem]:grid-cols-[16rem_minmax(0,1fr)]",
    ]}
>
    <div
        id="app-overlays"
        class="pointer-events-none fixed inset-0 z-50"
        role="region"
        aria-label="Application overlays"
        bind:this={overlayTarget}
    ></div>
    <header class="relative z-40 col-span-full flex items-center gap-4 border-b border-app-border bg-app-card px-4 shadow-sm">
        <button
            type="button"
            class="btn-icon hover:preset-tonal text-app-accent-strong [&_svg]:size-[1.2rem]"
            aria-label={navigationToggled ? "Close navigation" : "Open navigation"}
            aria-expanded={navigationToggled}
            aria-controls="primary-navigation"
            onclick={() => (navigationToggled = !navigationToggled)}
        >
            {#if navigationToggled}
                <XIcon aria-hidden="true" />
            {:else}
                <MenuIcon aria-hidden="true" />
            {/if}
        </button>

        <a class="inline-flex items-center rounded" href={resolve("/")} aria-label="Konfidence home">
            <img class="h-[1.4rem] w-[8.5rem] object-contain min-[52rem]:w-[10.5rem]" src={logoSrc} alt="Konfidence" />
        </a>

        <div class="chip bg-primary-100 text-primary-800 pointer-events-none">Skeleton</div>

        <div class="ml-auto flex items-center gap-2 [&_svg]:size-[1.2rem]">
            <button class="btn-icon hover:preset-tonal relative text-app-accent-strong" type="button" aria-label="Notifications, 3 unread">
                <BellIcon aria-hidden="true" />
                <span class="absolute -top-[0.2rem] -right-[0.2rem] grid h-4 min-w-4 place-items-center rounded-full border-2 border-app-card bg-app-error px-[0.2rem] text-[0.625rem] font-bold text-white" aria-hidden="true">3</span>
            </button>
            <Menu
                aria-label="Account menu"
                positioning={{ placement: "bottom-end", gutter: 10 }}
                onSelect={(details) => selectAccountAction(details.value)}
            >
                <Menu.Trigger
                    class="grid cursor-pointer rounded-full border-0 bg-transparent p-0"
                    aria-label={`Open account menu for ${data.user.name}`}
                >
                    <Avatar class="size-8 bg-app-accent/18 font-[650] text-app-accent-strong">
                        <Avatar.Fallback>{avatarInitials}</Avatar.Fallback>
                    </Avatar>
                </Menu.Trigger>
                <Portal target={overlayTarget}>
                    <Menu.Positioner class="pointer-events-auto z-50">
                        <Menu.Content class="w-[min(18rem,calc(100vw-2rem))] border border-app-border bg-app-card p-2 text-app-text shadow-2xl [&_[data-part=item]]:flex [&_[data-part=item]]:w-full [&_[data-part=item]]:cursor-pointer [&_[data-part=item]]:items-center [&_[data-part=item]]:rounded-[0.4rem] [&_[data-part=item]]:px-3 [&_[data-part=item]]:py-[0.7rem] [&_[data-part=item]]:text-app-text [&_[data-part=item][data-highlighted]]:bg-app-accent/10 [&_[data-part=item-text]]:flex [&_[data-part=item-text]]:items-center [&_[data-part=item-text]]:gap-[0.65rem] [&_svg]:size-4">
                            <Menu.ItemGroup>
                                <Menu.ItemGroupLabel class="grid gap-[0.15rem] border-b border-app-border px-3 pt-[0.7rem] pb-[0.9rem] text-app-text [&_span]:[overflow-wrap:anywhere] [&_span]:text-[0.82rem] [&_span]:text-app-muted">
                                    <strong>{data.user.name}</strong>
                                    <span>{data.user.email}</span>
                                </Menu.ItemGroupLabel>
                                <Menu.Item value="settings">
                                    <Menu.ItemText>
                                        <SettingsIcon aria-hidden="true" /> Settings
                                    </Menu.ItemText>
                                </Menu.Item>
                                <Menu.Item value="sign-out">
                                    <Menu.ItemText>
                                        <LogOutIcon aria-hidden="true" /> Sign Out
                                    </Menu.ItemText>
                                </Menu.Item>
                            </Menu.ItemGroup>
                        </Menu.Content>
                    </Menu.Positioner>
                </Portal>
            </Menu>
        </div>
    </header>

    <button
        class={[
            "fixed inset-x-0 top-[3.35rem] bottom-0 z-30 border-0 bg-black/42 transition-[opacity,visibility] duration-150 min-[52rem]:hidden",
            navigationToggled ? "visible opacity-100" : "invisible opacity-0",
        ]}
        type="button"
        aria-label="Close navigation"
        onclick={() => (navigationToggled = false)}
    ></button>

    <aside
        id="primary-navigation"
        class={[
            "fixed top-[3.35rem] bottom-0 left-0 z-35 min-h-0 w-[min(18rem,84vw)] overflow-y-auto border-r border-app-border bg-app-sidebar shadow-2xl transition-[transform,visibility] duration-150 min-[52rem]:relative min-[52rem]:inset-auto min-[52rem]:z-20 min-[52rem]:row-start-2 min-[52rem]:w-auto min-[52rem]:translate-x-0 min-[52rem]:visible min-[52rem]:shadow-none",
            navigationToggled ? "visible translate-x-0" : "invisible -translate-x-[105%]",
        ]}
        aria-label="Project navigation"
    >
        <ProjectSelector
            projects={data.projects}
            selectedProjectId={selectedProject?.id}
            onselect={selectProject}
            collapsed={navigationToggled}
        />
        {#if navigation.length > 0}
            <nav class="px-2 py-[0.8rem]" aria-label="Delivery">
                <h2 class={["m-0 px-3 py-[0.6rem] text-[0.78rem] font-[650] tracking-[0.06em] text-app-muted uppercase", navigationToggled && "min-[52rem]:hidden"]}>Delivery</h2>
                <ul class="m-0 grid list-none gap-[0.2rem] p-0">
                    {#each navigation as item (item.href)}
                        <li>
                            <a
                                class={[
                                    "flex min-h-11 items-center gap-[0.7rem] rounded-[0.4rem] border-l-[3px] border-transparent px-3 text-app-text no-underline hover:bg-app-accent/7 [&_svg]:size-[1.1rem] [&_svg]:shrink-0",
                                    navigationToggled && "min-[52rem]:justify-center min-[52rem]:px-0",
                                    page.url.pathname.startsWith(item.href) && "border-l-app-accent bg-app-accent/12 font-semibold text-app-accent-strong",
                                ]}
                                href={item.href}
                                aria-label={item.label}
                                aria-current={page.url.pathname.startsWith(item.href) ? "page" : undefined}
                                onclick={() => (navigationToggled = false)}
                            >
                                <item.icon aria-hidden="true" />
                                <span class={navigationToggled ? "min-[52rem]:hidden" : undefined}>{item.label}</span>
                            </a>
                        </li>
                    {/each}
                </ul>
            </nav>
        {/if}
    </aside>

    <main id="main-content" class="col-start-1 row-start-2 flex min-h-0 min-w-0 overflow-auto min-[52rem]:col-start-2">
        {@render children()}
    </main>
</div>

<SettingsDialog
    open={settingsOpen}
    tab={settingsTab ?? DEFAULT_SETTINGS_TAB}
    user={data.user}
    onClose={closeSettings}
    onTabChange={changeSettingsTab}
/>
