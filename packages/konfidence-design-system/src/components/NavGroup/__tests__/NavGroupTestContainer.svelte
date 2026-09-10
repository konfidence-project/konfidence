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
        <NavItem href="/landscape" active icon="landscape">Landscape</NavItem>
        <NavItem href="/vectors" icon="vectors">Vector Deployments</NavItem>
        <NavItem href="/artifacts" icon="artifacts" badge={2}>Artifact Deployments</NavItem>
    </NavGroup>
</div>
