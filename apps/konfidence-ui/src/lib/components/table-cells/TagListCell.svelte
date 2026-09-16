<script lang="ts">
    interface Props {
        labels: readonly string[];
        keys?: readonly string[];
        mono?: boolean;
    }
    let { labels, keys, mono = false }: Props = $props();
</script>

<div class="inline-flex flex-wrap gap-1">
    {#if labels.length === 0}
        <span class="text-[color:var(--text-tertiary)]">—</span>
    {:else}
        {#each labels as label, index (keys?.[index] ?? label)}
            <span
                class={[
                    "inline-flex items-center rounded-[var(--tag-radius)] bg-[color:var(--surface-sunken)] px-2 py-0.5",
                    "text-[length:var(--text-meta)] font-medium text-[color:var(--text-secondary)]",
                    mono && "font-[family-name:var(--font-mono)]",
                ]}>{label}</span
            >
        {/each}
    {/if}
</div>
