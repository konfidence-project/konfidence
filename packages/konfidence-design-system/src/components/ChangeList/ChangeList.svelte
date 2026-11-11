<script lang="ts">
    import TogglePill from "../TogglePill/TogglePill.svelte";
    import type { ChangeListItem } from "./types.js";

    interface Props {
        items: readonly ChangeListItem[];
        ariaLabel?: string;
        emptyLabel?: string;
        class?: string;
    }
    let { items, ariaLabel = "Changes", emptyLabel = "No changes.", class: className }: Props = $props();
    const LABELS = { added: "Added", changed: "Changed", removed: "Removed" } as const;
</script>

<div class={["change-list-container min-w-0", className]}>
    {#if items.length === 0}
        <p class="empty-hint mt-2 mb-0 text-body text-content-tertiary italic">{emptyLabel}</p>
    {:else}
        <ul class="change-list m-0 flex list-none flex-col gap-0.5 p-0" aria-label={ariaLabel}>
            {#each items as item, index (item.id)}
                <li>
                    {#if item.group && (index === 0 || items[index - 1].group !== item.group)}
                        <div class="change-group-label px-3 pt-3 pb-1 text-[10px] font-semibold tracking-[0.06em] text-content-tertiary uppercase">{item.group}</div>
                    {/if}
                    <div class={["change-row grid items-center gap-x-2 gap-y-1 rounded-[var(--radius-sm)] border-l-[3px] px-3 py-2 text-body", `change-row--${item.state}`]} data-state={item.state}>
                        <span class="change-badge shrink-0 whitespace-nowrap rounded-[var(--radius-sm)] px-[5px] py-0.5 text-[9px] leading-[1.2] font-bold tracking-[0.04em] uppercase">{item.label ?? LABELS[item.state]}</span>
                        <span class="change-name min-w-0 truncate font-mono" title={item.name}>{item.name}</span>
                        {#if (item.state !== "added" && item.oldValue !== undefined) || (item.state !== "removed" && item.newValue !== undefined)}
                            <span class="change-val flex min-w-0 flex-wrap items-center justify-end gap-1 text-meta">
                                {#if item.state !== "added" && item.oldValue !== undefined}
                                    <span class="change-val-old min-w-0 text-content-tertiary [overflow-wrap:anywhere]">
                                        <span class="sr-only">Previous value: </span>
                                        {#if typeof item.oldValue === "boolean"}<TogglePill on={item.oldValue} />{:else}{item.oldValue}{/if}
                                    </span>
                                {/if}
                                <span class="change-val-arrow text-content-tertiary" aria-hidden="true">→</span>
                                {#if item.state !== "removed" && item.newValue !== undefined}
                                    <span class="change-val-new min-w-0 font-medium text-content-primary [overflow-wrap:anywhere]">
                                        <span class="sr-only">New value: </span>
                                        {#if typeof item.newValue === "boolean"}<TogglePill on={item.newValue} />{:else}{item.newValue}{/if}
                                    </span>
                                {/if}
                            </span>
                        {/if}
                    </div>
                </li>
            {/each}
        </ul>
    {/if}
</div>

<style>
    .change-list-container { container-type: inline-size; }
    .change-row {
        grid-template-columns: auto minmax(0, 1fr);
        border-left-color: var(--change-color);
        background: color-mix(in srgb, var(--change-color) 8%, transparent);
    }
    .change-row--added { --change-color: #19a765; }
    .change-row--changed { --change-color: var(--amber-400); }
    .change-row--removed {
        --change-color: #d32020;
        background: color-mix(in srgb, var(--change-color) 7%, transparent);
    }
    .change-badge { background: var(--change-color); color: #fff; }
    .change-row--changed .change-badge { color: #5c3000; }
    .change-val { grid-column: 1 / -1; }
    @container (min-width: 380px) {
        .change-row { grid-template-columns: auto minmax(0, 1fr) minmax(0, auto); }
        .change-val { grid-column: 3; max-width: 220px; }
    }
</style>
