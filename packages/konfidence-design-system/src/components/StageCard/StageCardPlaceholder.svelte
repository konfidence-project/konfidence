<script lang="ts">
    interface Props {
        title: string;
        landscapeName?: string;
        message?: string;
        ariaLabel?: string;
    }

    let {
        title,
        landscapeName,
        message = "No stages yet",
        ariaLabel,
    }: Props = $props();
</script>

<!-- The empty counterpart of StageCard: it reuses the same card frame and
     header so a stage-less landscape sits in the grid with the same shape as a
     populated stage, distinguished by a dashed border and no interactivity. -->
<div
    class="box-border flex h-64 w-full flex-col gap-3 rounded-[var(--card-radius)] border border-dashed border-[var(--border-subtle)] bg-[var(--surface-card)] p-4 text-[var(--text-primary)]"
    data-testid="stage-card-placeholder"
    aria-label={ariaLabel}
>
    <header class="flex items-baseline gap-3">
        <div class="flex min-w-0 flex-col gap-0.5">
            {#if landscapeName}
                <span
                    class="truncate text-[length:var(--text-meta)] font-semibold uppercase tracking-wide text-[var(--text-tertiary)]"
                    title={landscapeName}
                >
                    {landscapeName}
                </span>
            {/if}
            <h4
                class="m-0 truncate text-[length:var(--text-body)] font-bold"
                {title}
            >
                {title}
            </h4>
        </div>
    </header>

    <div
        class="flex flex-1 items-center justify-center text-center text-[length:var(--text-sm)] text-[var(--text-secondary)]"
    >
        {message}
    </div>
</div>
