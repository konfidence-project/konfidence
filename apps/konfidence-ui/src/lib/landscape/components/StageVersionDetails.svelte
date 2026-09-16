<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import { stageStatuses } from "$lib/landscape/stageStatus";
    import type { StageVersion } from "$lib/landscape/stageStatus";

    interface Props {
        label: string;
        version?: StageVersion;
    }
    let { label, version }: Props = $props();
    const titleId = $props.id();
</script>

<section
    class="min-w-0 rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-card)] p-5"
    aria-labelledby={titleId}
>
    <h2 class="mt-0 text-[length:var(--text-h3)]" id={titleId}>
        {label} version
    </h2>
    {#if version}
        <StatusBadge status={stageStatuses[version.status].badge}
            >{stageStatuses[version.status].label}</StatusBadge
        >
        <dl class="grid gap-2 text-[length:var(--text-sm)]">
            <dt class="text-[var(--text-tertiary)]">Version ID</dt>
            <dd class="m-0 mb-3 font-[family-name:var(--font-mono)] [overflow-wrap:anywhere]">{version.id}</dd>
            <dt class="text-[var(--text-tertiary)]">Vector</dt>
            <dd class="m-0 mb-3 font-[family-name:var(--font-mono)] [overflow-wrap:anywhere]">{version.vector}</dd>
            <dt class="text-[var(--text-tertiary)]">Generation</dt>
            <dd class="m-0 mb-3 font-[family-name:var(--font-mono)] [overflow-wrap:anywhere]">{version.stageGeneration}</dd>
        </dl>
    {:else}<p>No {label.toLowerCase()} version yet.</p>{/if}
</section>
