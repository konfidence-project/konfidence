<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import { stageStatuses } from "$lib/landscape/stageStatus";
    import type { StageVersion } from "$lib/konfidence-api/types";

    interface Props {
        label: string;
        version?: StageVersion;
    }
    let { label, version }: Props = $props();
    const titleId = $props.id();
</script>

<section
    class="min-w-0 rounded-lg border border-outline-subtle bg-surface-card p-5"
    aria-labelledby={titleId}
>
    <h2 class="mt-0 text-h3" id={titleId}>
        {label} version
    </h2>
    {#if version}
        <StatusBadge status={stageStatuses[version.status].badge}
            >{stageStatuses[version.status].label}</StatusBadge
        >
        <dl class="grid gap-2 text-compact">
            <dt class="text-content-tertiary">Version ID</dt>
            <dd class="m-0 mb-3 font-mono [overflow-wrap:anywhere]">{version.id}</dd>
            <dt class="text-content-tertiary">Vector</dt>
            <dd class="m-0 mb-3 font-mono [overflow-wrap:anywhere]">{version.vector}</dd>
            <dt class="text-content-tertiary">Generation</dt>
            <dd class="m-0 mb-3 font-mono [overflow-wrap:anywhere]">{version.stageGeneration}</dd>
        </dl>
    {:else}<p>No {label.toLowerCase()} version yet.</p>{/if}
</section>
