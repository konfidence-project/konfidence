<script lang="ts">
    import NavItem from "../NavItem.svelte";

    interface Props {
        label: string;
        href?: string;
        active?: boolean;
        icon?: string;
        badge?: number;
    }

    let {
        label,
        href = "/landscape",
        active = false,
        icon,
        badge,
    }: Props = $props();
</script>

<div style="padding: 12px; background: var(--surface-default); width: 220px;">
    <NavItem {href} {active} {icon} {badge} data-testid="nav-item">{label}</NavItem>
</div>
