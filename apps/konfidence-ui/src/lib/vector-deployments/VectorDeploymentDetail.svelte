<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import type { VectorDeploymentRow } from "./deployments.js";
    import { statusLabel, statusTone } from "./deployments.js";

    interface Props {
        row: VectorDeploymentRow;
    }

    let { row }: Props = $props();
</script>

<div class="flex flex-col gap-5">
    <section class="flex flex-col gap-2" aria-label="VectorDeployment">
        <h3
            class="m-0 text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]"
        >
            Vector deployment
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
                data-testid="vectordeployment-detail-id"
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

    <section class="flex flex-col gap-2" aria-label="Stage">
        <h3
            class="m-0 text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]"
        >
            Stage
        </h3>

        <p class="m-0 flex flex-col gap-0.5 text-[length:var(--text-sm)] text-[color:var(--text-primary)]">
            <span>{row.stageName}</span>
            {#if row.stageName !== row.stageId}
                <span class="font-[family-name:var(--font-mono)] text-[color:var(--text-tertiary)]">{row.stageId}</span>
            {/if}
        </p>
    </section>
</div>
