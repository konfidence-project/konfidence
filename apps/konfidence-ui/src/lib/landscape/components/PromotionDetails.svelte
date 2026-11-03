<script lang="ts">
    import type { VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
    import type { StageEdge } from "#lib/landscape/landscapeGraph.ts";

    interface Props {
        promotion: StageEdge;
        config?: VectorPromotionConfig;
    }
    let { promotion, config }: Props = $props();

    const columnClass =
        "flex min-w-0 flex-col gap-3 rounded-[var(--card-radius)] border border-[color:var(--border-subtle)] bg-surface-subtle p-4";
    const columnTitleClass =
        "m-0 text-meta font-semibold uppercase tracking-[0.03em] text-content-tertiary";
</script>

<div class="flex flex-col gap-4" data-testid="promotion-details">
    <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <section class={columnClass} aria-labelledby="promotion-artifacts-title">
            <h3 id="promotion-artifacts-title" class={columnTitleClass}>Artifact Changes</h3>
            <p class="m-0 text-compact text-content-tertiary">
                Not yet implemented.
            </p>
        </section>

        <section class={columnClass} aria-labelledby="promotion-toggles-title">
            <h3 id="promotion-toggles-title" class={columnTitleClass}>Feature Toggles</h3>
            <p class="m-0 text-compact text-content-tertiary">
                Not yet implemented.
            </p>
        </section>

        <section class={columnClass} aria-labelledby="promotion-info-title">
            <h3 id="promotion-info-title" class={columnTitleClass}>Info</h3>
            <dl
                class="m-0 grid grid-cols-[auto_1fr] gap-x-3 gap-y-2 text-compact"
            >
                <dt class="text-content-tertiary">Source</dt>
                <dd class="m-0 min-w-0 truncate">{promotion.sourceName}</dd>
                <dt class="text-content-tertiary">Target</dt>
                <dd class="m-0 min-w-0 truncate">{promotion.targetName}</dd>
                {#if config}
                    <dt class="text-content-tertiary">Config</dt>
                    <dd class="m-0 min-w-0 truncate font-mono">{config.id}</dd>
                {/if}
            </dl>
        </section>
    </div>
</div>
