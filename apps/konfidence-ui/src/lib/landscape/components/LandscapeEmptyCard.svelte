<script lang="ts">
    interface Props {
        landscapeName: string;
    }
    let { landscapeName }: Props = $props();
</script>

<!-- A landscape with no stages still appears on the canvas as a muted
     placeholder so it stays visible and discoverable. -->
<div
    class="box-border flex h-64 w-full flex-col gap-3 rounded-[var(--card-radius)] border border-dashed border-[var(--border-subtle)] bg-[var(--surface-card)] p-4 text-[var(--text-primary)]"
    data-testid="landscape-empty-card"
    aria-label={`Landscape ${landscapeName} has no stages yet`}
>
    <header class="flex flex-col gap-0.5">
        <span
            class="truncate text-[length:var(--text-meta)] font-semibold uppercase tracking-wide text-[var(--text-tertiary)]"
            title={landscapeName}
        >
            {landscapeName}
        </span>
    </header>
    <div
        class="flex flex-1 items-center justify-center text-center text-[length:var(--text-sm)] text-[var(--text-secondary)]"
    >
        No stages yet
    </div>
</div>
