<script lang="ts">
    import favicon from "$lib/assets/favicon.svg";
    import "@ui5/webcomponents/dist/Button.js";
    import "@ui5/webcomponents/dist/Text.js";
    import "@ui5/webcomponents/dist/Title.js";
    import "@ui5/webcomponents/dist/ToggleButton.js";
    import "@ui5/webcomponents/dist/Avatar.js";

    import "@ui5/webcomponents-fiori/dist/NavigationLayout.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigation.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigationGroup.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigationItem.js";
    import "@ui5/webcomponents-fiori/dist/SideNavigationSubItem.js";
    import "@ui5/webcomponents-fiori/dist/ShellBar.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarBranding.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarSearch.js";
    import "@ui5/webcomponents-fiori/dist/ShellBarItem.js";

    import "@ui5/webcomponents-icons/dist/home.js";
    import "@ui5/webcomponents-icons/dist/group.js";
    import "@ui5/webcomponents-icons/dist/locate-me.js";
    import "@ui5/webcomponents-icons/dist/calendar.js";
    import "@ui5/webcomponents-icons/dist/history.js";
    import "@ui5/webcomponents-icons/dist/source-code.js";
    import "@ui5/webcomponents-icons/dist/background.js";
    import "@ui5/webcomponents-icons/dist/activity-assigned-to-goal.js";
    import "@ui5/webcomponents-icons/dist/action-settings.js";
    import "@ui5/webcomponents-icons/dist/chain-link.js";
    import "@ui5/webcomponents-icons/dist/document-text.js";
    import "@ui5/webcomponents-icons/dist/compare.js";
    import "@ui5/webcomponents-icons/dist/locked.js";
    import "@ui5/webcomponents-icons/dist/menu2.js";
    import "@ui5/webcomponents-icons/dist/sys-help.js";
    import "@ui5/webcomponents-icons/dist/customer.js";
    import "@ui5/webcomponents-icons/dist/da.js";

    let { children } = $props();
</script>

<svelte:head>
    <link rel="icon" href={favicon} />
</svelte:head>

<ui5-navigation-layout id="nl1">
    <ui5-shellbar slot="header" notifications-count="72" show-notifications>
        <ui5-button icon="menu2" slot="startButton" id="startButton"
        ></ui5-button>
        <ui5-shellbar-branding slot="branding">
            Product Identifier
            <img
                slot="logo"
                src="https://ui5.github.io/webcomponents/images/sap-logo-svg.svg"
            />
        </ui5-shellbar-branding>
        <ui5-shellbar-search
            slot="searchField"
            show-clear-icon
            placeholder="Search Apps, Products"
        ></ui5-shellbar-search>

        <ui5-shellbar-item icon="sys-help" text="Help"></ui5-shellbar-item>
        <ui5-toggle-button icon="sap-icon://da" tooltip="Joule" slot="assistant"
        ></ui5-toggle-button>
        <ui5-avatar slot="profile">
            <img
                src="https://ui5.github.io/webcomponents/images/avatars/man_avatar_3.png"
            />
        </ui5-avatar>
    </ui5-shellbar>
    <ui5-side-navigation id="sn1" slot="sideContent">
        <!-- Items -->
        <ui5-side-navigation-item text="Home" href="#home" icon="home"
        ></ui5-side-navigation-item>
        <ui5-side-navigation-group text="Group 1" expanded>
            <ui5-side-navigation-item
                text="Item 1"
                href="#item1"
                icon="locate-me"
                expanded
            >
                <ui5-side-navigation-sub-item text="Sub Item 1" href="#subitem1"
                ></ui5-side-navigation-sub-item>
                <ui5-side-navigation-sub-item text="Sub Item 2" href="#subitem2"
                ></ui5-side-navigation-sub-item>
            </ui5-side-navigation-item>
            <ui5-side-navigation-item
                text="Item 2"
                href="#item2"
                icon="calendar"
                expanded
            >
                <ui5-side-navigation-sub-item text="Sub Item 3" href="#subitem3"
                ></ui5-side-navigation-sub-item>
                <ui5-side-navigation-sub-item text="Sub Item 4" href="#subitem4"
                ></ui5-side-navigation-sub-item>
            </ui5-side-navigation-item>
            <ui5-side-navigation-item
                text="Item 3"
                href="#item2"
                icon="activity-assigned-to-goal"
                expanded
            >
                <ui5-side-navigation-sub-item text="Sub Item 5" href="#subitem5"
                ></ui5-side-navigation-sub-item>
                <ui5-side-navigation-sub-item text="Sub Item 6" href="#subitem6"
                ></ui5-side-navigation-sub-item>
            </ui5-side-navigation-item>
        </ui5-side-navigation-group>
        <ui5-side-navigation-group text="Group 2" expanded>
            <ui5-side-navigation-item text="Item 4" href="#item4" icon="history"
            ></ui5-side-navigation-item>
            <ui5-side-navigation-item
                text="Item 5"
                href="#item5"
                icon="source-code"
            ></ui5-side-navigation-item>
            <ui5-side-navigation-item
                text="Item 6"
                href="#item6"
                icon="background"
            ></ui5-side-navigation-item>
        </ui5-side-navigation-group>
        <!-- Fixed Items -->
        <ui5-side-navigation-item
            slot="fixedItems"
            text="Legal"
            href="https://www.sap.com/about/legal/impressum.html"
            target="_blank"
            unselectable
            icon="compare"
        ></ui5-side-navigation-item>
        <ui5-side-navigation-item
            slot="fixedItems"
            text="Privacy"
            href="https://www.sap.com/about/legal/privacy.html"
            target="_blank"
            unselectable
            icon="locked"
        ></ui5-side-navigation-item>
        <ui5-side-navigation-item
            slot="fixedItems"
            text="Terms of Use"
            href="https://www.sap.com/terms-of-use"
            target="_blank"
            unselectable
            icon="document-text"
        ></ui5-side-navigation-item>
    </ui5-side-navigation>
    <div class="content">
        {@render children()}
    </div>
</ui5-navigation-layout>
