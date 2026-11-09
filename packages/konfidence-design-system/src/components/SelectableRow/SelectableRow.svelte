<script lang="ts">
    import type { Snippet } from "svelte";
    import { getGroupSelection } from "../ExpandableGroup/selection-context.js";

    interface Props {
        ariaLabel: string;
        selected?: boolean;
        /** Opt into the enclosing group's single selection. Unique within that group. */
        value?: string;
        disabled?: boolean;
        label?: string;
        metadata?: Snippet;
        /** Noninteractive content to the right, inside the selection button. */
        trailing?: Snippet;
        /** Align desktop content with an expandable header without an outer gutter. */
        inset?: boolean;
        onselect?: () => void;
        /** Noninteractive phrasing content; put controls in actions instead. */
        children?: Snippet;
        /** Independent controls outside the selection button. */
        actions?: Snippet;
        /** Optional focus-restoration target for the selection button. */
        id?: string;
        class?: string;
    }

    let {
        ariaLabel,
        selected = false,
        value,
        disabled = false,
        label,
        metadata,
        trailing,
        inset = false,
        onselect,
        children,
        actions,
        id,
        class: className,
    }: Props = $props();
    const group = getGroupSelection();
    const isSelected = $derived(value !== undefined && group ? group.selectedValue === value : selected);
    const select = () => {
        if (disabled) {
            return;
        }
        if (value !== undefined && group && !group.select(value)) {
            return;
        }
        onselect?.();
    };
</script>

<div class={["selectable-row relative flex min-w-0 flex-wrap items-center text-compact leading-normal text-content-primary transition-colors duration-[var(--motion-fast)] motion-reduce:transition-none", !disabled && !isSelected && "hover:bg-surface-subtle", isSelected && "selectable-row--selected bg-selection-bg", className]}>
    <button {id} {disabled} type="button" aria-label={ariaLabel} aria-pressed={isSelected} onclick={select}
        class={["relative flex min-h-16 min-w-0 grow basis-64 items-center gap-3 p-4 text-left cursor-pointer focus-visible:z-10 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-outline-focus disabled:cursor-not-allowed disabled:opacity-50 [overflow-wrap:anywhere]", inset && "sm:pl-10"]}>
        <span class="min-w-0 grow">
            {#if children}{@render children()}{:else}<span class="block font-semibold leading-[1.3]">{label ?? ariaLabel}</span>{/if}
            {#if metadata}<span class="mt-1 flex flex-wrap items-center gap-2 text-content-secondary">{@render metadata()}</span>{/if}
        </span>
        {#if trailing}<span class="flex shrink-0 items-center gap-2">{@render trailing()}</span>{/if}
    </button>
    {#if actions}
        <div class="flex min-w-0 flex-wrap items-center gap-3 px-4 py-3">{@render actions()}</div>
    {/if}
</div>

<style>
    .selectable-row--selected::before {
        content: "";
        position: absolute;
        top: -1px;
        bottom: 0;
        left: 0;
        width: 3px;
        background: var(--gradient-amber);
        pointer-events: none;
    }
</style>
