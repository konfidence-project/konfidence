<script lang="ts">
    import "@ui5/webcomponents/dist/Avatar.js";
    import "@ui5/webcomponents/dist/Label.js";
    import "@ui5/webcomponents/dist/Tag.js";
    import "@ui5/webcomponents/dist/Title.js";
    import "@ui5/webcomponents-fiori/dist/UserSettingsAppearanceView.js";
    import "@ui5/webcomponents-fiori/dist/UserSettingsAppearanceViewGroup.js";
    import "@ui5/webcomponents-fiori/dist/UserSettingsAppearanceViewItem.js";
    import "@ui5/webcomponents-fiori/dist/UserSettingsDialog.js";
    import "@ui5/webcomponents-fiori/dist/UserSettingsItem.js";
    import "@ui5/webcomponents-fiori/dist/UserSettingsView.js";

    import {
        DEFAULT_SETTINGS_TAB,
        SETTINGS_TABS,
        type SettingsTab,
    } from "./settings-tab.js";
    import { selectTheme, themePreference, themes } from "$lib/stores/theme.svelte";
    import type { AuthUser } from "$lib/auth/types";
    import { STAGE_CARD_VARIANTS } from "$lib/components/stage/variants.js";
    import { getStageCardVariantPreference } from "$lib/stores/stage-card-variant.svelte";
    import { t } from "$lib/stores/i18n.svelte";

    type UserSettingsItemSelectEventDetail =
        import("@ui5/webcomponents-fiori/dist/UserSettingsDialog.js").UserSettingsItemSelectEventDetail;
    type UserSettingsAppearanceViewItemSelectEventDetail =
        import("@ui5/webcomponents-fiori/dist/UserSettingsAppearanceView.js").UserSettingsAppearanceViewItemSelectEventDetail;

    const AVATAR_INITIALS_MAX = 2;

    interface Props {
        onClose: () => void;
        onTabChange: (tab: SettingsTab) => void;
        open: boolean;
        tab: SettingsTab;
        user: AuthUser;
    }

    const { onClose, onTabChange, open, tab, user }: Props = $props();

    const activeTab: SettingsTab = $derived(tab ?? DEFAULT_SETTINGS_TAB);

    const stageCardVariant = getStageCardVariantPreference();

    const avatarInitials = $derived(
        user.name
            .split(" ")
            .map((part) => part[0] ?? "")
            .join("")
            .slice(0, AVATAR_INITIALS_MAX)
            .toUpperCase(),
    );

    const accountSubtitle = $derived(user.email ?? t("SETTINGS_PROFILE_FALLBACK_SUBTITLE"));
    const accountDescription = $derived(user.roles.join(", "));

    const themeLabelKey = (themeId: string): string => {
        if (themeId === "konfidence") {
            return "THEME_KONFIDENCE";
        }
        if (themeId === "konfidence-dark") {
            return "THEME_KONFIDENCE_DARK";
        }
        if (themeId === "sap_horizon") {
            return "THEME_SAP_HORIZON";
        }
        return themeId;
    };

    interface AccountDetail {
        label: string;
        value: string;
    }

    const accountDetails: AccountDetail[] = $derived([
        { label: t("SETTINGS_PROFILE_FIELD_EMAIL"), value: user.email },
        { label: t("SETTINGS_PROFILE_FIELD_GIVEN_NAME"), value: user.givenName },
        { label: t("SETTINGS_PROFILE_FIELD_FAMILY_NAME"), value: user.familyName },
        { label: t("SETTINGS_PROFILE_FIELD_ROLES"), value: accountDescription || t("SETTINGS_PROFILE_ROLES_NONE") },
    ]);

    const handleClose = (): void => {
        onClose();
    };

    const handleSelectionChange = (
        event: CustomEvent<UserSettingsItemSelectEventDetail>,
    ): void => {
        const nextTab = event.detail.item.getAttribute("data-tab-id");
        if (!nextTab) {
            return;
        }
        const found = SETTINGS_TABS.find((definition) => definition.id === nextTab);
        if (!found) {
            return;
        }
        if (found.id === activeTab) {
            return;
        }
        onTabChange(found.id);
    };

    const handleAppearanceChange = (
        event: CustomEvent<UserSettingsAppearanceViewItemSelectEventDetail>,
    ): void => {
        const key = event.detail.item.getAttribute("item-key");
        if (!key) {
            return;
        }
        selectTheme(key);
    };

    const handleStageCardVariantChange = (
        event: CustomEvent<UserSettingsAppearanceViewItemSelectEventDetail>,
    ): void => {
        const key = event.detail.item.getAttribute("item-key");
        if (!key) {
            return;
        }
        stageCardVariant.select(key);
    };
</script>

<ui5-user-settings-dialog
    id="settings-dialog"
    header-text={t("SETTINGS_HEADER")}
    {open}
    onui5-close={handleClose}
    onui5-selection-change={handleSelectionChange}
>
    <ui5-user-settings-item
        icon="user-settings"
        text={t("SETTINGS_TAB_PROFILE")}
        tooltip={t("SETTINGS_TAB_PROFILE")}
        header-text={t("SETTINGS_TAB_PROFILE")}
        selected={activeTab === "profile"}
        data-tab-id="profile"
    >
        <ui5-user-settings-view>
            <div class="profile-view">
                <div class="profile-header">
                    <ui5-avatar
                        class="profile-avatar"
                        initials={avatarInitials}
                        color-scheme="Accent6"
                        size="XL"
                        accessible-name={t("SETTINGS_PROFILE_AVATAR_ARIA", user.name)}
                    ></ui5-avatar>
                    <ui5-title class="profile-title" level="H3" size="H4">
                        {user.name}
                    </ui5-title>
                    <span class="profile-subtitle">{accountSubtitle}</span>
                    {#if accountDescription}
                        <span class="profile-description">{accountDescription}</span>
                    {/if}
                </div>

                <div class="account-details" aria-label={t("SETTINGS_PROFILE_DETAILS_ARIA")}>
                    <dl class="account-fields">
                        {#each accountDetails as field (field.label)}
                            <div class="account-row">
                                <dt>{field.label}</dt>
                                <dd>{field.value}</dd>
                            </div>
                        {/each}
                    </dl>
                </div>
            </div>
        </ui5-user-settings-view>
    </ui5-user-settings-item>

    <ui5-user-settings-item
        icon="palette"
        text={t("SETTINGS_TAB_APPEARANCE")}
        tooltip={t("SETTINGS_TAB_APPEARANCE")}
        header-text={t("SETTINGS_TAB_APPEARANCE")}
        selected={activeTab === "appearance"}
        data-tab-id="appearance"
    >
        <ui5-user-settings-appearance-view
            text={t("SETTINGS_APPEARANCE_THEME")}
            onui5-selection-change={handleAppearanceChange}
        >
            <ui5-user-settings-appearance-view-group header-text={t("SETTINGS_APPEARANCE_GROUP_KONFIDENCE")}>
                {#each themes.filter((option) => option.id.startsWith("konfidence")) as option (option.id)}
                    <ui5-user-settings-appearance-view-item
                        item-key={option.id}
                        text={t(themeLabelKey(option.id))}
                        icon={option.id.includes("dark") ? "dark-mode" : "light-mode"}
                        selected={themePreference.selected === option.id}
                    ></ui5-user-settings-appearance-view-item>
                {/each}
            </ui5-user-settings-appearance-view-group>
            <ui5-user-settings-appearance-view-group header-text={t("SETTINGS_APPEARANCE_GROUP_SAP")}>
                {#each themes.filter((option) => !option.id.startsWith("konfidence")) as option (option.id)}
                    <ui5-user-settings-appearance-view-item
                        item-key={option.id}
                        text={t(themeLabelKey(option.id))}
                        icon={option.id.includes("dark") ? "dark-mode" : "light-mode"}
                        selected={themePreference.selected === option.id}
                    ></ui5-user-settings-appearance-view-item>
                {/each}
            </ui5-user-settings-appearance-view-group>
        </ui5-user-settings-appearance-view>
    </ui5-user-settings-item>

    <ui5-user-settings-item
        icon="upstacked-chart"
        text={t("SETTINGS_TAB_LANDSCAPE")}
        tooltip={t("SETTINGS_TAB_LANDSCAPE")}
        header-text={t("SETTINGS_TAB_LANDSCAPE")}
        selected={activeTab === "landscape"}
        data-tab-id="landscape"
    >
        <ui5-user-settings-appearance-view
            text={t("SETTINGS_LANDSCAPE_STAGE_CARDS")}
            onui5-selection-change={handleStageCardVariantChange}
        >
            <ui5-user-settings-appearance-view-group header-text={t("SETTINGS_LANDSCAPE_CARD_STYLE")}>
                {#each STAGE_CARD_VARIANTS as variant (variant.id)}
                    <ui5-user-settings-appearance-view-item
                        item-key={variant.id}
                        text={t(variant.labelKey)}
                        selected={stageCardVariant.selected === variant.id}
                    ></ui5-user-settings-appearance-view-item>
                {/each}
            </ui5-user-settings-appearance-view-group>
        </ui5-user-settings-appearance-view>
    </ui5-user-settings-item>
</ui5-user-settings-dialog>

<style>
    .profile-view {
        display: grid;
        gap: 1.25rem;
        padding: 0.25rem 0;
    }

    .profile-header {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.25rem;
    }

    .profile-avatar {
        max-width: 7rem;
        max-height: 7rem;
    }

    .profile-title {
        margin-top: 0.25rem;
        text-align: center;
    }

    .profile-subtitle {
        text-align: center;
        color: var(--sapContent_LabelColor);
        font-family: var(--sapFontFamily);
        font-size: var(--sapFontSize);
        overflow-wrap: anywhere;
    }

    .profile-description {
        text-align: center;
        color: var(--sapContent_LabelColor);
        font-family: var(--sapFontFamily);
        font-size: var(--sapFontSize);
    }

    .account-details {
        padding: 0.25rem 0;
    }

    .account-fields {
        display: grid;
        gap: 0.5rem;
        margin: 0;
    }

    .account-row {
        display: grid;
        grid-template-columns: minmax(9rem, 12rem) 1fr;
        gap: 1rem;
        align-items: baseline;
    }

    dt {
        margin: 0;
        color: var(--sapContent_LabelColor);
        font-size: var(--sapFontSmallSize);
    }

    dd {
        margin: 0;
        color: var(--sapTextColor);
        font-family: var(--sapFontFamily);
        overflow-wrap: anywhere;
    }

    @media (max-width: 42rem) {
        .account-row {
            grid-template-columns: 1fr;
            gap: 0.15rem;
        }
    }
</style>
