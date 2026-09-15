<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import type { ArtifactDeploymentRow } from "./deployments.js";
    import { statusLabel, statusTone } from "./deployments.js";

    interface Props {
        row: ArtifactDeploymentRow;
    }

    let { row }: Props = $props();
</script>

<div class="flex flex-col gap-5">
    <section class="flex flex-col gap-2" aria-label="Artifact">
        <h3
            class="m-0 text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]"
        >
            Artifact
        </h3>
        <dl class="m-0 grid grid-cols-[minmax(0,8rem)_1fr] gap-x-3 gap-y-2 text-[length:var(--text-sm)]">
            <dt class="text-[color:var(--text-tertiary)]">Component</dt>
            <dd class="m-0 break-words text-[color:var(--text-primary)]">{row.component}</dd>
            <dt class="text-[color:var(--text-tertiary)]">Version</dt>
            <dd class="m-0 break-words font-[family-name:var(--font-mono)] text-[color:var(--text-primary)]">
                {row.version}
            </dd>
            <dt class="text-[color:var(--text-tertiary)]">Repository</dt>
            <dd class="m-0 break-words font-[family-name:var(--font-mono)] text-[color:var(--text-primary)]">
                {row.repository}
            </dd>
            <dt class="text-[color:var(--text-tertiary)]">Status</dt>
            <dd class="m-0 text-[color:var(--text-primary)]">
                <StatusBadge status={statusTone(row.status)}>{statusLabel(row.status)}</StatusBadge>
            </dd>
            <dt class="text-[color:var(--text-tertiary)]">Deployment id</dt>
            <dd
                class="m-0 break-words font-[family-name:var(--font-mono)] text-[color:var(--text-primary)]"
                data-testid="artifact-detail-id"
            >
                {row.id}
            </dd>
        </dl>
    </section>

    <section class="flex flex-col gap-2" aria-label="Landscape">
        <h3
            class="m-0 text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]"
        >
            Landscape
        </h3>
        <p class="m-0 flex flex-col gap-0.5 text-[length:var(--text-sm)] text-[color:var(--text-primary)]">
            <span>{row.landscape}</span>
            {#if row.landscape !== row.landscapeId}
                <span class="font-[family-name:var(--font-mono)] text-[color:var(--text-tertiary)]">{row.landscapeId}</span>
            {/if}
        </p>
    </section>

    <section class="flex flex-col gap-2" aria-label="Stages">
        <h3
            class="m-0 text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]"
        >
            Stages ({row.stageNames.length})
        </h3>
        {#if row.stageNames.length === 0}
            <p class="m-0 text-[length:var(--text-sm)] text-[color:var(--text-tertiary)]">
                No stages linked.
            </p>
        {:else}
            <ul class="m-0 flex flex-col gap-2 p-0 list-none" data-testid="artifact-detail-stages">
                {#each row.stageNames as name, index (row.stageIds[index] ?? name)}
                    <li
                        class="rounded-[var(--radius-md)] border border-[color:var(--border-subtle)] bg-[color:var(--surface-subtle)] px-3 py-2"
                    >
                        <div class="flex flex-col items-start gap-1">
                            <span>{name}</span>
                            {#if name !== row.stageIds[index]}
                                <span class="font-[family-name:var(--font-mono)] text-[color:var(--text-tertiary)]"
                                    >{row.stageIds[index]}</span
                                >
                            {/if}
                        </div>
                    </li>
                {/each}
            </ul>
        {/if}
    </section>

    <section class="flex flex-col gap-2" aria-label="Vector deployments">
        <h3
            class="m-0 text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]"
        >
            Vector deployments ({row.vectorDeploymentIds.length})
        </h3>
        {#if row.vectorDeploymentIds.length === 0}
            <p class="m-0 text-[length:var(--text-sm)] text-[color:var(--text-tertiary)]">
                No vector deployments linked.
            </p>
        {:else}
            <ul class="m-0 flex flex-col gap-2 p-0 list-none" data-testid="artifact-detail-vectors">
                {#each row.vectorDeploymentIds as id, index (id)}
                    {@const related = row.relatedVectorDeployments.find((vector) => vector.id === id)}
                    <li
                        class="rounded-[var(--radius-md)] border border-[color:var(--border-subtle)] bg-[color:var(--surface-subtle)] px-3 py-2"
                    >
                        <div class="flex flex-col items-start gap-1">
                            <span class="font-[family-name:var(--font-mono)]">{id}</span>
                            {#if related}
                                <span
                                    >{related.vector.componentName}@{related.vector.componentVersion}</span
                                >
                                <StatusBadge status="deploying">{related.status}</StatusBadge>
                            {:else if row.vectorDeploymentLabels[index] && row.vectorDeploymentLabels[index] !== id}
                                <span>{row.vectorDeploymentLabels[index]}</span>
                            {/if}
                        </div>
                    </li>
                {/each}
            </ul>
        {/if}
    </section>
</div>
