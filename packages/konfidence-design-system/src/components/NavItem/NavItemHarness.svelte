<script lang="ts">
    import NavItem from "./NavItem.svelte";
    import type { IconName } from "../Icon/icons.js";

    interface Props {
        label: string;
        href?: string;
        active?: boolean;
        icon?: IconName;
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
