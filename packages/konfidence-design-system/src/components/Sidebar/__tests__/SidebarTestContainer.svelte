<script lang="ts">
    import NavGroup from "../../NavGroup/NavGroup.svelte";
    import NavItem from "../../NavItem/NavItem.svelte";
    import ProjectSelection from "../../ProjectSelection/ProjectSelection.svelte";
    import Sidebar from "../Sidebar.svelte";

    interface Props {
        withFooter?: boolean;
        withMobileSwitcher?: boolean;
    }

    let { withFooter = true, withMobileSwitcher = false }: Props = $props();
</script>

<div data-testid="sidebar-root" style="width: 224px; height: 400px;">
    <Sidebar>
        {#snippet mobileSwitcher()}
            {#if withMobileSwitcher}
                <ProjectSelection name="Konfidence" />
            {/if}
        {/snippet}
        {#snippet footer()}
            {#if withFooter}<span data-testid="sidebar-footer-text">v1.2.3 · eu-west-1</span>{/if}
        {/snippet}
        <NavGroup label="Delivery">
            <NavItem href="/landscape" active icon="grid">Landscape</NavItem>
            <NavItem href="/vectors" icon="chain-link">Vector Deployments</NavItem>
            <NavItem href="/artifacts" icon="product">Artifact Deployments</NavItem>
        </NavGroup>
    </Sidebar>
</div>
