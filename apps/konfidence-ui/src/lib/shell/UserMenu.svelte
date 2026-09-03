<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { Menu } from "@konfidence/design-system/components";
    import { useSession } from "$lib/auth/session.svelte";

    /**
     * Avatar trigger + user menu. Skeleton's `Menu.Trigger` renders a real
     * `<button>` when no `element` snippet is provided; we style it with the
     * `.avatar` class vocabulary from the design system so the visual stays
     * consistent while getting native keyboard activation (Enter/Space) for
     * free.
     * Sign-out is dispatched through the menu's `onSelect` so the existing
     * sign-out route handles the session flow without menu-specific coupling.
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
        class="avatar avatar--orbit"
        aria-label={`Open user menu for ${session.user?.name ?? "current user"}`}
        data-testid="user-menu-trigger"
    >
        {initials}
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
