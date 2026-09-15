<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { Avatar, Menu } from "@konfidence/design-system/components";
    import { useSession } from "$lib/auth/session.svelte";

    /**
     * Avatar trigger + user menu. Follows the same idiom as
     * `ProjectSelector.svelte`: `Menu.Trigger` delegates its DOM to the
     * `element` snippet so the real DS component (`<Avatar>`) becomes the
     * trigger and inherits Zag's keyboard/focus/aria wiring via the
     * spread `attributes`. Sign-out is dispatched through the menu's
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
    <Menu.Trigger>
        {#snippet element(attributes)}
            <Avatar
                {...attributes}
                class={attributes.class}
                {initials}
                orbit
                ariaLabel={`Open user menu for ${session.user?.name ?? "current user"}`}
                data-testid="user-menu-trigger"
            />
        {/snippet}
    </Menu.Trigger>
    <Menu.Positioner>
        <Menu.Content header>
            <Menu.Header>
                {#snippet name()}
                    <span data-testid="user-menu-name">{session.user?.name ?? ""}</span>
                {/snippet}
                {#snippet mail()}
                    {session.user?.email ?? ""}
                {/snippet}
            </Menu.Header>
            <Menu.Separator />
            <Menu.Item value={SIGN_OUT_VALUE} variant="danger" data-testid="sign-out">
                Sign out
            </Menu.Item>
        </Menu.Content>
    </Menu.Positioner>
</Menu>
