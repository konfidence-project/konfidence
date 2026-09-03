<script lang="ts">
    import { Menu } from "./index.js";

    interface Props {
        variant: "plain" | "with-header" | "size-sm" | "size-lg";
    }

    let { variant }: Props = $props();

    const size = $derived.by((): "sm" | "md" | "lg" => {
        if (variant === "size-sm") {
            return "sm";
        }
        if (variant === "size-lg") {
            return "lg";
        }
        return "md";
    });
    const isHeader = $derived(variant === "with-header");

    // Menu is always open for screenshotting the Content surface.
    let menuOpen = $state(true);
</script>

<Menu
    open={menuOpen}
    onOpenChange={(details) => {
        menuOpen = details.open;
    }}
>
    <Menu.Trigger data-testid="menu-trigger" class="btn btn--secondary">
        Open
    </Menu.Trigger>
    <Menu.Positioner>
        <Menu.Content {size} header={isHeader} data-testid="menu-root">
            {#if isHeader}
                <Menu.Header initials="AA">
                    {#snippet name()}Alex Admin{/snippet}
                    {#snippet mail()}alex.admin@example.com{/snippet}
                </Menu.Header>
                <Menu.Separator />
                <Menu.Item value="sign-out" variant="danger">Sign out</Menu.Item>
            {:else}
                <Menu.ItemGroup>
                    <Menu.Label>Project</Menu.Label>
                    <Menu.Item value="a">Aurora</Menu.Item>
                    <Menu.Item value="k" active>Konfidence</Menu.Item>
                </Menu.ItemGroup>
                <Menu.Separator />
                <Menu.Item value="delete" variant="danger">Delete project</Menu.Item>
            {/if}
        </Menu.Content>
    </Menu.Positioner>
</Menu>
