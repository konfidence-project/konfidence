<script lang="ts">
    import NavGroup from "../NavGroup/NavGroup.svelte";
    import NavItem from "../NavItem/NavItem.svelte";
    import ProjectSwitcher from "../ProjectSwitcher/ProjectSwitcher.svelte";
    import Sidebar from "./Sidebar.svelte";

    interface Props {
        withFooter?: boolean;
        withMobileSwitcher?: boolean;
    }

    let { withFooter = true, withMobileSwitcher = false }: Props = $props();
</script>

<div data-testid="sidebar-root" style="width: 224px; height: 400px;">
    <Sidebar>
        {#snippet mobileSwitcher()}
            {#if withMobileSwitcher}<ProjectSwitcher name="Konfidence" />{/if}
        {/snippet}
        {#snippet footer()}
            {#if withFooter}<span data-testid="sidebar-footer-text">v1.2.3 · eu-west-1</span>{/if}
        {/snippet}
        <NavGroup label="Delivery">
            <NavItem href="/landscape" active icon="landscape">Landscape</NavItem>
            <NavItem href="/vectors" icon="vectors">Vector Deployments</NavItem>
            <NavItem href="/artifacts" icon="artifacts">Artifact Deployments</NavItem>
        </NavGroup>
    </Sidebar>
</div>
