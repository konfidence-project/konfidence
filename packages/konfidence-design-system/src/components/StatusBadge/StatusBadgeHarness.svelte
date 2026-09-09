<script lang="ts">
    import StatusBadge from "./StatusBadge.svelte";

    interface Props {
        status: string;
        showDot?: boolean;
        label: string;
    }

    let { label, ...rest }: Props = $props();
</script>

<StatusBadge {...rest}>{label}</StatusBadge>
