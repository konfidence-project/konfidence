<script lang="ts">
    import TogglePill from "../TogglePill/TogglePill.svelte";
    import type { ChangeListItem, ChangeState } from "./types.js";

    interface Props {
        items: readonly ChangeListItem[];
        ariaLabel?: string;
        emptyLabel?: string;
        class?: string;
    }
    let { items, ariaLabel = "Changes", emptyLabel = "No changes.", class: className }: Props = $props();
    const LABELS = { added: "Added", changed: "Changed", removed: "Removed" } as const;
    const STATE_CLASSES: Record<ChangeState, { badge: string; row: string }> = {
        added: { badge: "bg-status-healthy-bg text-status-healthy-fg", row: "border-l-status-healthy-solid bg-status-healthy-solid/8" },
        changed: { badge: "bg-status-warning-bg text-status-warning-fg", row: "border-l-status-warning-solid bg-status-warning-solid/8" },
        removed: { badge: "bg-status-error-bg text-status-error-fg", row: "border-l-status-error-solid bg-status-error-solid/7" },
    };
</script>

<div class={["change-list-container @container min-w-0", className]}>
    {#if items.length === 0}
        <p class="empty-hint mt-2 mb-0 text-body text-content-tertiary italic">{emptyLabel}</p>
    {:else}
        <ul class="change-list m-0 flex list-none flex-col gap-0.5 p-0" aria-label={ariaLabel}>
            {#each items as item, index (item.id)}
                <li>
                    {#if item.group && (index === 0 || items[index - 1].group !== item.group)}
                        <div class="change-group-label px-3 pt-3 pb-1 text-[10px] font-semibold tracking-[0.06em] text-content-tertiary uppercase">{item.group}</div>
                    {/if}
                    <div class={["change-row grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2 gap-y-1 rounded-[var(--radius-sm)] border-l-[3px] px-3 py-2 text-body @[380px]:grid-cols-[auto_minmax(0,1fr)_minmax(0,auto)]", `change-row--${item.state}`, STATE_CLASSES[item.state].row]} data-state={item.state}>
                        <span class={["change-badge shrink-0 whitespace-nowrap rounded-[var(--radius-sm)] px-[5px] py-0.5 text-[9px] leading-[1.2] font-bold tracking-[0.04em] uppercase", STATE_CLASSES[item.state].badge]}>{item.label ?? LABELS[item.state]}</span>
                        <span class="change-name min-w-0 truncate font-mono" title={item.name}>{item.name}</span>
                        {#if (item.state !== "added" && item.oldValue !== undefined) || (item.state !== "removed" && item.newValue !== undefined)}
                            <span class="change-val col-span-full flex min-w-0 flex-wrap items-center justify-end gap-1 text-meta @[380px]:col-start-3 @[380px]:col-end-auto @[380px]:max-w-[220px]">
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
