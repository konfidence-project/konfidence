<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { Menu } from "@skeletonlabs/skeleton-svelte";
    import { useSession } from "$lib/auth/session.svelte";

    /**
     * Avatar trigger + user menu. Sign-out is dispatched through the menu's
     * `onSelect` so the existing sign-out route handles the session flow
     * without menu-specific coupling.
     */
    const AVATAR_INITIALS_MAX = 2;
    const SIGN_OUT_VALUE = "sign-out";

    const session = useSession();

    const initials = $derived(
        (session.user?.name ?? "?")
            .split(" ")
            .map((part) => part[0] ?? "")
            .join("")
            .slice(0, AVATAR_INITIALS_MAX)
            .toUpperCase() || "?",
    );

    const handleSelect = (details: { value: string }): void => {
        if (details.value === SIGN_OUT_VALUE) {
            void goto(resolve("/logout"));
        }
    };
</script>

<Menu positioning={{ placement: "bottom-end" }} onSelect={handleSelect}>
    <Menu.Trigger
        class="avatar"
        data-testid="user-menu-trigger"
        aria-label={`Open user menu for ${session.user?.name ?? "current user"}`}
    >
        {initials}
    </Menu.Trigger>
    <Menu.Positioner class="menu-positioner">
        <Menu.Content class="menu menu--header">
            <div class="menu__header">
                <div class="menu__header-info">
                    <div class="menu__header-name" data-testid="user-menu-name">
                        {session.user?.name ?? ""}
                    </div>
                    {#if session.user?.email}
                        <div class="menu__header-mail">{session.user.email}</div>
                    {/if}
                </div>
            </div>
            <Menu.Separator class="menu__sep" />
            <Menu.Item
                value={SIGN_OUT_VALUE}
                class="menu__item menu__item--danger"
                data-testid="sign-out"
            >
                <span class="menu__text">Sign out</span>
            </Menu.Item>
        </Menu.Content>
    </Menu.Positioner>
</Menu>
