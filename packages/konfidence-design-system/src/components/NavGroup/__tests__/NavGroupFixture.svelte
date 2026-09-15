<script lang="ts">
    import NavGroup from "../NavGroup.svelte";
    import NavItem from "../../NavItem/NavItem.svelte";

    interface Props {
        label?: string;
        hideLabel?: boolean;
    }

    let { label = "Delivery", hideLabel = false }: Props = $props();
</script>

<div
    data-testid="nav-group-root"
    style="padding: 12px; background: var(--surface-default); width: 220px;"
>
    <NavGroup label={hideLabel ? undefined : label}>
        <NavItem href="/landscape" active icon="grid">Landscape</NavItem>
        <NavItem href="/vectors" icon="chain-link">Vector Deployments</NavItem>
        <NavItem href="/artifacts" icon="product" badge={2}>Artifact Deployments</NavItem>
    </NavGroup>
</div>
