<script lang="ts">
    import Avatar from "../Avatar.svelte";

    interface Props {
        initials?: string;
        orbit?: boolean;
    }

    let { initials = "AA", orbit = false }: Props = $props();
</script>

<div style="padding: 16px; background: var(--surface-default);">
    <Avatar {initials} {orbit} ariaLabel="Alex Admin" />
</div>
