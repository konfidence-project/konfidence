<script lang="ts">
    import IconButton from "./IconButton.svelte";
    import type { IconName } from "../Icon/icons.js";

    interface Props {
        icon?: IconName;
        badge?: number;
        ariaLabel?: string;
    }

    let { icon = "bell", badge, ariaLabel = "Notifications" }: Props = $props();
</script>

<div style="padding: 16px; background: var(--surface-default);">
    <IconButton {icon} {ariaLabel} {badge} />
</div>
