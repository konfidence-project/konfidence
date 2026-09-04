<script lang="ts">
    import StatusBadge from "./StatusBadge.svelte";

    interface Props {
        status:
            | "healthy"
            | "warning"
            | "degraded"
            | "error"
            | "promoting"
            | "deploying"
            | "queued";
        showDot?: boolean;
        label: string;
    }

    let { label, ...rest }: Props = $props();
</script>

<StatusBadge {...rest}>{label}</StatusBadge>
