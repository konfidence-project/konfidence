<script lang="ts">
    import type { VectorDeploymentRow } from "./deployments.js";
    import VectorDeploymentFields from "./VectorDeploymentFields.svelte";

    interface Props {
        row: VectorDeploymentRow;
    }

    let { row }: Props = $props();
</script>

<div class="flex flex-col gap-5">
    <VectorDeploymentFields vectorDeployment={row.raw} />

    <section class="flex flex-col gap-2" aria-label="Landscape">
        <h3
            class="m-0 text-meta font-semibold uppercase tracking-[0.03em] text-content-tertiary"
        >
            Landscape
        </h3>
        <p class="m-0 flex flex-col gap-0.5 text-compact text-content-primary">
            <span>{row.landscape}</span>
            {#if row.landscape !== row.landscapeId}
                <span class="font-[family-name:var(--font-mono)] text-[color:var(--text-tertiary)]">{row.landscapeId}</span>
            {/if}
        </p>
    </section>

    <section class="flex flex-col gap-2" aria-label="Stage">
        <h3
            class="m-0 text-meta font-semibold uppercase tracking-[0.03em] text-content-tertiary"
        >
            Stage
        </h3>

        <p class="m-0 flex flex-col gap-0.5 text-compact text-content-primary">
            <span>{row.stageName}</span>
            {#if row.stageName !== row.stageId}
                <span class="font-[family-name:var(--font-mono)] text-[color:var(--text-tertiary)]">{row.stageId}</span>
            {/if}
        </p>
    </section>
</div>
