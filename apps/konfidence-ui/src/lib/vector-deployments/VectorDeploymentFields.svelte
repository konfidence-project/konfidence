<script lang="ts">
    import { StatusBadge } from "@konfidence/design-system/components";
    import type { VectorDeployment } from "#lib/konfidence-api/types.ts";
    import { statusLabel, statusTone } from "./deployments.js";

    interface Props {
        vectorDeployment: VectorDeployment;
    }

    let { vectorDeployment }: Props = $props();
</script>

<section class="flex flex-col gap-2" aria-label="VectorDeployment">
    <h3
        class="m-0 text-meta font-semibold uppercase tracking-[0.03em] text-content-tertiary"
    >
        Vector deployment
    </h3>
    <dl class="m-0 grid grid-cols-[minmax(0,8rem)_1fr] gap-x-3 gap-y-2 text-compact">
        <dt class="text-[color:var(--text-tertiary)]">Component</dt>
        <dd class="m-0 break-words text-[color:var(--text-primary)]">
            {vectorDeployment.vector.componentName}
        </dd>
        <dt class="text-[color:var(--text-tertiary)]">Version</dt>
        <dd class="m-0 break-words font-[family-name:var(--font-mono)] text-[color:var(--text-primary)]">
            {vectorDeployment.vector.componentVersion}
        </dd>
        <dt class="text-[color:var(--text-tertiary)]">Repository</dt>
        <dd class="m-0 break-words font-[family-name:var(--font-mono)] text-[color:var(--text-primary)]">
            {vectorDeployment.vector.repository}
        </dd>
        <dt class="text-[color:var(--text-tertiary)]">Status</dt>
        <dd class="m-0 text-[color:var(--text-primary)]">
            <StatusBadge status={statusTone(vectorDeployment.status)}>
                {statusLabel(vectorDeployment.status)}
            </StatusBadge>
        </dd>
        <dt class="text-[color:var(--text-tertiary)]">Deployment id</dt>
        <dd
            class="m-0 break-words font-[family-name:var(--font-mono)] text-[color:var(--text-primary)]"
            data-testid="vectordeployment-detail-id"
        >
            {vectorDeployment.id}
        </dd>
    </dl>
</section>
