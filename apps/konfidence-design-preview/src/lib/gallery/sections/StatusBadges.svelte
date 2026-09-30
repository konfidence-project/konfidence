<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    const STATUSES = [
        "healthy",
        "warning",
        "degraded",
        "error",
        "promoting",
        "deploying",
        "queued",
    ] as const;
</script>

<Section
    id="status-badges"
    title="Status badges"
    subtitle="Double-coded (color + dot + text) via the real <StatusBadge> component."
    status="implemented"
>
    <Sample title="All statuses">
        {#each STATUSES as status (status)}
            <StatusBadge {status}>{status}</StatusBadge>
        {/each}
    </Sample>

    <Sample title="Unknown fallback">
        <StatusBadge status="mystery">Unknown</StatusBadge>
    </Sample>
</Section>
