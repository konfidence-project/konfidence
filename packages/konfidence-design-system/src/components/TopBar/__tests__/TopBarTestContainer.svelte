<script lang="ts">
    import Avatar from "../../Avatar/Avatar.svelte";
    import ProjectSelection from "../../ProjectSelection/ProjectSelection.svelte";
    import TopBar from "../TopBar.svelte";

    interface Props {
        withHamburger?: boolean;
    }

    let { withHamburger = false }: Props = $props();
</script>

<div data-testid="topbar-root">
    <TopBar onHamburger={withHamburger ? () => undefined : undefined}>
        {#snippet logo()}
            <span
                class="topbar__logo"
                style="display: inline-flex; align-items: center; font-weight: var(--weight-bold); color: var(--text-primary);"
                >Konfidence</span
            >
        {/snippet}
        {#snippet switcher()}
            <ProjectSelection name="Konfidence" />
        {/snippet}
        {#snippet actions()}
            <Avatar initials="AA" orbit ariaLabel="User menu" role="button" tabindex={0} />
        {/snippet}
    </TopBar>
</div>
