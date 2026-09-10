<script lang="ts">
    import IconButton from "../IconButton.svelte";

    interface Props {
        icon?: string;
        badge?: number;
        ariaLabel?: string;
    }

    let { icon = "bell", badge, ariaLabel = "Notifications" }: Props = $props();
</script>

<div style="padding: 16px; background: var(--surface-default);">
    <IconButton {icon} {ariaLabel} {badge} />
</div>
