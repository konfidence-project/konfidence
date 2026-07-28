<script lang="ts">
    import PaletteIcon from "@lucide/svelte/icons/palette";
    import UserRoundIcon from "@lucide/svelte/icons/user-round";
    import * as Avatar from "$lib/components/ui/avatar/index.js";
    import * as Dialog from "$lib/components/ui/dialog/index.js";
    import { Label } from "$lib/components/ui/label/index.js";
    import * as NativeSelect from "$lib/components/ui/native-select/index.js";
    import * as RadioGroup from "$lib/components/ui/radio-group/index.js";
    import * as Tabs from "$lib/components/ui/tabs/index.js";
    import type { AuthUser } from "$lib/auth/types";
    import { getThemePreference } from "$lib/stores/theme.svelte";
    import { themeModes } from "$lib/theme";
    import { m } from "$lib/paraglide/messages.js";
    import { getLocale, isLocale, setLocale } from "$lib/paraglide/runtime.js";
    import { MediaQuery } from "svelte/reactivity";
    import {
        DEFAULT_SETTINGS_TAB,
        type SettingsTab,
        parseSettingsTab,
    } from "./settings-tab.js";

    interface Props {
        onClose: () => void;
        onTabChange: (tab: SettingsTab) => void;
        open: boolean;
        returnFocus?: () => void;
        tab: SettingsTab;
        user: AuthUser;
    }

    const AVATAR_INITIALS_MAX = 2;
    const { onClose, onTabChange, open, returnFocus, tab, user }: Props = $props();
    const theme = getThemePreference();
    const compactSettings = new MediaQuery("(max-width: 42rem)", false);
    const activeTab = $derived(tab ?? DEFAULT_SETTINGS_TAB);
    const tabOrientation = $derived.by(() => {
        if (compactSettings.current) {
            return "horizontal" as const;
        }
        return "vertical" as const;
    });
    const avatarInitials = $derived(
        user.name
            .split(" ")
            .map((part) => part[0] ?? "")
            .join("")
            .slice(0, AVATAR_INITIALS_MAX)
            .toUpperCase(),
    );
    const roles = $derived(user.roles.join(", ") || m.settings_none());
    const accountDetails = $derived([
        { label: m.settings_email(), value: user.email },
        { label: m.settings_given_name(), value: user.givenName },
        { label: m.settings_family_name(), value: user.familyName },
        { label: m.settings_roles(), value: roles },
    ]);

    const changeTab = (value: string): void => {
        const nextTab = parseSettingsTab(value);
        if (nextTab && nextTab !== activeTab) {
            onTabChange(nextTab);
        }
    };

    const handleOpenChange = (nextOpen: boolean): void => {
        if (!nextOpen && open) {
            onClose();
        }
    };

    const restoreFocus = (event: Event): void => {
        if (returnFocus) {
            event.preventDefault();
            returnFocus();
        }
    };

    const changeLocale = (event: Event): void => {
        const locale = (event.currentTarget as HTMLSelectElement).value;
        if (isLocale(locale) && locale !== getLocale()) {
            void setLocale(locale);
        }
    };
</script>

<Dialog.Root {open} onOpenChange={handleOpenChange}>
    <Dialog.Content
        class="max-h-[calc(100dvh-2rem)] grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-x-hidden overflow-y-auto p-0 sm:max-w-4xl max-[42rem]:h-[calc(100dvh-1rem)] max-[42rem]:max-w-[calc(100%-1rem)]"
        onCloseAutoFocus={restoreFocus}
    >
        <Dialog.Header class="border-b px-6 pt-6 pb-4">
            <Dialog.Title>{m.account_settings()}</Dialog.Title>
            <Dialog.Description>{m.settings_description()}</Dialog.Description>
        </Dialog.Header>

        <Tabs.Root
            value={activeTab}
            onValueChange={changeTab}
            orientation={tabOrientation}
            class="grid min-h-[34rem] w-full min-w-0 grid-cols-[14rem_minmax(0,1fr)] max-[42rem]:flex max-[42rem]:min-h-0 max-[42rem]:flex-col"
        >
            <Tabs.List
                class="!h-full !w-full min-w-0 flex-col items-stretch justify-start gap-[0.35rem] rounded-none border-r bg-muted p-4 max-[42rem]:!h-auto max-[42rem]:w-full max-[42rem]:flex-row max-[42rem]:overflow-x-auto max-[42rem]:border-r-0 max-[42rem]:border-b"
                aria-label={m.settings_sections()}
            >
                <Tabs.Trigger class="h-10 flex-none justify-start text-foreground" value="profile">
                    <UserRoundIcon /> {m.settings_profile()}
                </Tabs.Trigger>
                <Tabs.Trigger class="h-10 flex-none justify-start text-foreground" value="appearance">
                    <PaletteIcon /> {m.settings_appearance()}
                </Tabs.Trigger>
            </Tabs.List>

            <Tabs.Content
                value="profile"
                class="m-0 w-full min-w-0 px-8 pt-6 pb-8 max-[42rem]:p-5"
            >
                <h2 class="mb-6 text-[1.35rem] font-semibold">{m.settings_profile()}</h2>
                <div class="mx-auto mb-8 flex max-w-lg items-center gap-5">
                    <Avatar.Root class="size-24">
                        <Avatar.Fallback class="text-3xl">{avatarInitials}</Avatar.Fallback>
                    </Avatar.Root>
                    <div>
                        <h3 class="text-xl font-semibold">{user.name}</h3>
                        <p class="text-muted-foreground">{user.email}</p>
                        <p class="text-muted-foreground">{roles}</p>
                    </div>
                </div>
                <dl class="mx-auto grid max-w-lg gap-3" aria-label={m.settings_identity_details()}>
                    {#each accountDetails as field (field.label)}
                        <div class="grid grid-cols-[9rem_1fr] gap-4 max-[42rem]:grid-cols-1 max-[42rem]:gap-[0.1rem]">
                            <dt class="text-[0.8rem] text-muted-foreground">{field.label}</dt>
                            <dd class="m-0 [overflow-wrap:anywhere]">{field.value}</dd>
                        </div>
                    {/each}
                </dl>
            </Tabs.Content>

            <Tabs.Content
                value="appearance"
                class="m-0 w-full min-w-0 px-8 pt-6 pb-8 max-[42rem]:p-5"
            >
                <h2 class="mb-6 text-[1.35rem] font-semibold">{m.settings_appearance()}</h2>
                <p class="-mt-4 mb-6 text-muted-foreground">
                    {m.settings_appearance_description()}
                </p>
                <div class="mb-6 grid gap-2 border-b pb-6">
                    <Label for="settings-language">{m.settings_language()}</Label>
                    <p class="text-sm text-muted-foreground">
                        {m.settings_language_description()}
                    </p>
                    <NativeSelect.Root
                        class="w-full max-w-sm"
                        id="settings-language"
                        value={getLocale()}
                        onchange={changeLocale}
                    >
                        <NativeSelect.Option value="en">{m.language_english()}</NativeSelect.Option>
                        <NativeSelect.Option value="de">{m.language_german()}</NativeSelect.Option>
                    </NativeSelect.Root>
                </div>
                <RadioGroup.Root
                    value={theme.selected}
                    onValueChange={(value) => theme.select(value)}
                    aria-label={m.settings_theme()}
                    class="gap-3"
                >
                    {#each themeModes as option (option.id)}
                        <div
                            class={[
                                "flex min-w-0 items-center gap-3 rounded-xl border p-3",
                                theme.selected === option.id &&
                                    "border-primary ring-2 ring-primary/20",
                            ]}
                        >
                            <RadioGroup.Item value={option.id} id={`theme-${option.id}`} />
                            <Label
                                class="flex min-w-0 flex-1 cursor-pointer items-center gap-3"
                                for={`theme-${option.id}`}
                            >
                                <span
                                    class={[
                                        "h-9 w-14 shrink-0 rounded-[0.65rem] border",
                                        option.id === "konfidence" &&
                                            "bg-[linear-gradient(135deg,#139cc7_0_52%,#ffaa00_52%)]",
                                        option.id === "konfidence-dark" &&
                                            "bg-[linear-gradient(135deg,#111_0_52%,#80d2e0_52%)]",
                                        option.id === "sap_horizon" &&
                                            "bg-[linear-gradient(135deg,#fff_0_52%,#0a6ed1_52%)]",
                                        option.id === "system" &&
                                            "bg-[linear-gradient(135deg,#f5f6f7_0_52%,#111_52%)]",
                                    ]}
                                ></span>
                                <span class="grid min-w-0 gap-[0.15rem]">
                                    <strong>{option.label}</strong>
                                    <small class="text-xs text-muted-foreground">
                                        {option.description}
                                    </small>
                                </span>
                            </Label>
                        </div>
                    {/each}
                </RadioGroup.Root>
            </Tabs.Content>
        </Tabs.Root>
    </Dialog.Content>
</Dialog.Root>
