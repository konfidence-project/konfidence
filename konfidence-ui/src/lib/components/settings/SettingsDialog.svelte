<script lang="ts">
    import PaletteIcon from "@lucide/svelte/icons/palette";
    import UserIcon from "@lucide/svelte/icons/user";
    import XIcon from "@lucide/svelte/icons/x";
    import { Dialog, Portal, Tabs } from "@skeletonlabs/skeleton-svelte";

    import type { AuthUser } from "$lib/auth/types";
    import { getLocalePreference } from "$lib/locale-preference.svelte";
    import { locales } from "$lib/locale";
    import { getThemePreference } from "$lib/theme-preference.svelte";
    import { themes } from "$lib/theme";
    import type { SettingsTab } from "./settings-tab.js";

    const AVATAR_INITIALS_MAX = 2;

    interface Props {
        onClose: () => void;
        onTabChange: (tab: SettingsTab) => void;
        open: boolean;
        tab: SettingsTab;
        user: AuthUser;
    }

    const { onClose, onTabChange, open, tab, user }: Props = $props();
    const locale = getLocalePreference();
    const theme = getThemePreference();
    const initials = $derived(
        user.name
            .split(" ")
            .map((part) => part[0] ?? "")
            .join("")
            .slice(0, AVATAR_INITIALS_MAX)
            .toUpperCase(),
    );
    const accountDetails = $derived([
        { id: "email", label: locale.translate("settings.account.email"), value: user.email },
        { id: "given-name", label: locale.translate("settings.account.givenName"), value: user.givenName },
        { id: "family-name", label: locale.translate("settings.account.familyName"), value: user.familyName },
        { id: "roles", label: locale.translate("settings.account.roles"), value: user.roles.join(", ") || locale.translate("settings.account.noRoles") },
    ]);
</script>

<Dialog {open} onOpenChange={(details) => !details.open && onClose()}>
    <Portal>
        <Dialog.Backdrop class="fixed inset-0 z-60 bg-black/48 backdrop-blur-[2px]" />
        <Dialog.Positioner class="fixed inset-0 z-61 grid place-items-center p-4 max-[42rem]:p-0">
            <Dialog.Content class="card grid max-h-[min(42rem,calc(100vh-2rem))] w-[min(54rem,100%)] grid-rows-[auto_minmax(0,1fr)] overflow-hidden border border-app-border bg-app-card text-app-text shadow-2xl max-[42rem]:h-full max-[42rem]:max-h-none max-[42rem]:w-full max-[42rem]:rounded-none max-[42rem]:border-0">
                <header class="flex items-center justify-between border-b border-app-border px-[1.2rem] py-4">
                    <Dialog.Title class="text-[1.2rem] font-[650]">{locale.translate("settings.title")}</Dialog.Title>
                    <Dialog.CloseTrigger class="btn-icon hover:preset-tonal [&_svg]:size-[1.15rem]" aria-label={locale.translate("settings.close")}>
                        <XIcon aria-hidden="true" />
                    </Dialog.CloseTrigger>
                </header>
                <Dialog.Description class="sr-only">
                    {locale.translate("settings.description")}
                </Dialog.Description>

                <Tabs
                    value={tab}
                    orientation="vertical"
                    onValueChange={(details) => onTabChange(details.value as SettingsTab)}
                    class="grid min-h-0 grid-cols-[12rem_minmax(0,1fr)] max-[42rem]:grid-cols-1 max-[42rem]:grid-rows-[auto_minmax(0,1fr)]"
                >
                    <Tabs.List class="flex flex-col gap-1 border-r border-app-border bg-app-bg/70 px-[0.6rem] py-4 max-[42rem]:flex-row max-[42rem]:border-r-0 max-[42rem]:border-b" aria-label={locale.translate("settings.sections")}>
                        <Tabs.Trigger class="flex min-h-[2.7rem] cursor-pointer items-center justify-start gap-[0.65rem] rounded-[0.4rem] border-0 bg-transparent px-[0.8rem] text-app-text data-[selected]:bg-app-accent/14 data-[selected]:font-[650] data-[selected]:text-app-accent-strong max-[42rem]:flex-1 max-[42rem]:justify-center [&_svg]:size-[1.1rem]" value="profile">
                            <UserIcon aria-hidden="true" /> {locale.translate("settings.profile")}
                        </Tabs.Trigger>
                        <Tabs.Trigger class="flex min-h-[2.7rem] cursor-pointer items-center justify-start gap-[0.65rem] rounded-[0.4rem] border-0 bg-transparent px-[0.8rem] text-app-text data-[selected]:bg-app-accent/14 data-[selected]:font-[650] data-[selected]:text-app-accent-strong max-[42rem]:flex-1 max-[42rem]:justify-center [&_svg]:size-[1.1rem]" value="appearance">
                            <PaletteIcon aria-hidden="true" /> {locale.translate("settings.appearance")}
                        </Tabs.Trigger>
                        <Tabs.Indicator />
                    </Tabs.List>

                    <Tabs.Content value="profile" class="min-w-0 overflow-y-auto p-[clamp(1.25rem,4vw,2.5rem)]">
                        <div class="mb-8 grid justify-items-center gap-[0.3rem] text-center">
                            <span class="mb-[0.4rem] grid size-[5.25rem] place-items-center rounded-full bg-app-accent/18 text-[1.65rem] font-bold text-app-accent-strong" aria-label={locale.translate("settings.avatarFor", { name: user.name })}>{initials}</span>
                            <h2 class="m-0 text-[1.35rem]">{user.name}</h2>
                            <p class="m-0 [overflow-wrap:anywhere] text-app-muted">{user.email}</p>
                        </div>
                        <dl class="m-0 grid border-t border-app-border">
                            {#each accountDetails as detail (detail.id)}
                                <div class="grid grid-cols-[9rem_minmax(0,1fr)] gap-4 border-b border-app-border py-[0.8rem] max-[42rem]:grid-cols-1 max-[42rem]:gap-[0.2rem]">
                                    <dt class="text-[0.82rem] text-app-muted">{detail.label}</dt>
                                    <dd class="m-0 [overflow-wrap:anywhere]">{detail.value}</dd>
                                </div>
                            {/each}
                        </dl>
                    </Tabs.Content>

                    <Tabs.Content value="appearance" class="min-w-0 overflow-y-auto p-[clamp(1.25rem,4vw,2.5rem)]">
                        <h2 class="m-0 text-[1.35rem]">{locale.translate("settings.appearance")}</h2>
                        <p class="mt-[0.35rem] mb-7 text-app-muted">{locale.translate("settings.appearanceDescription")}</p>
                        <fieldset class="m-0 grid grid-cols-3 gap-3 border-0 p-0 max-[42rem]:grid-cols-1">
                            <legend class="mb-[0.7rem] font-[650]">{locale.translate("settings.theme")}</legend>
                            {#each themes as option (option.id)}
                                <label class={["relative grid cursor-pointer gap-[0.7rem] rounded-[0.65rem] border-2 border-app-border p-3 hover:border-app-accent max-[42rem]:grid-cols-[7rem_minmax(0,1fr)] max-[42rem]:items-center", theme.selected === option.id && "border-app-accent"]}>
                                    <input
                                        class="absolute top-[0.6rem] right-[0.6rem] accent-app-accent"
                                        type="radio"
                                        name="theme"
                                        value={option.id}
                                        checked={theme.selected === option.id}
                                        onchange={() => theme.select(option.id)}
                                    />
                                    <span data-theme={option.id} class="grid h-20 grid-cols-[28%_1fr] grid-rows-[24%_1fr] overflow-hidden rounded-[0.35rem] border border-app-border bg-app-bg max-[42rem]:h-16" aria-hidden="true">
                                        <i class="col-span-full border-b border-app-border bg-app-card"></i>
                                        <i class="border-r border-app-border bg-app-card"></i>
                                        <i class="m-[0.6rem] rounded-[0.2rem] bg-app-accent opacity-80"></i>
                                    </span>
                                    <span class="grid gap-[0.15rem]">
                                        <strong>{locale.translate(`settings.themes.${option.id}.label`)}</strong>
                                        <small class="text-app-muted">{locale.translate(`settings.themes.${option.id}.description`)}</small>
                                    </span>
                                </label>
                            {/each}
                        </fieldset>
                        <label class="mt-7 grid max-w-72 gap-2 font-[650]" for="locale-select">
                            {locale.translate("settings.language")}
                            <select
                                id="locale-select"
                                class="select border-app-border bg-app-card font-normal text-app-text"
                                value={locale.selected}
                                onchange={(event) => locale.select(event.currentTarget.value)}
                            >
                                {#each locales as option (option.id)}
                                    <option value={option.id}>{option.label}</option>
                                {/each}
                            </select>
                        </label>
                    </Tabs.Content>
                </Tabs>
            </Dialog.Content>
        </Dialog.Positioner>
    </Portal>
</Dialog>
