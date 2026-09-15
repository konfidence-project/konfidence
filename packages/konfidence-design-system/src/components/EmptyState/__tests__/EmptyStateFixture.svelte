<script lang="ts">
    import EmptyState from "../EmptyState.svelte";

    interface Props {
        tone?: "empty" | "info" | "error";
        title?: string;
        description?: string;
    }
    let {
        tone = "empty",
        title = "No artifact deployments",
        description = "This project does not have any artifact deployments yet.",
    }: Props = $props();
</script>

<div style="width: 320px; padding: 16px; background: var(--surface-canvas); color: var(--text-primary);">
    <EmptyState {tone} {title} {description}>
        {#snippet icon()}
            <svg
                width="28"
                height="28"
                viewBox="0 0 20 20"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
            >
                <ellipse cx="10" cy="4.5" rx="7" ry="2.5" />
                <path d="M3 4.5v11c0 1.4 3.1 2.5 7 2.5s7-1.1 7-2.5v-11" />
                <path d="M3 10c0 1.4 3.1 2.5 7 2.5s7-1.1 7-2.5" />
            </svg>
        {/snippet}
    </EmptyState>
</div>
