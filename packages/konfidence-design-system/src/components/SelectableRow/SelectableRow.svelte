<script lang="ts">
    import type { Snippet } from "svelte";
    import { getGroupSelection } from "../ExpandableGroup/selection-context.js";

    interface Props {
        ariaLabel: string;
        /** Unique within the enclosing group's selection scope. */
        value: string;
        /** Align desktop content with an expandable header without an outer gutter. */
        inset?: boolean;
        /** Noninteractive phrasing content; put controls in actions instead. */
        children: Snippet;
        /** Independent controls outside the selection button. */
        actions?: Snippet;
        /** Optional focus-restoration target for the selection button. */
        id?: string;
    }

    let {
        ariaLabel,
        value,
        inset = false,
        children,
        actions,
        id,
    }: Props = $props();
    const group = getGroupSelection();
    const isSelected = $derived(group.selectedValue === value);
</script>

<div class={["selectable-row relative flex min-w-0 flex-wrap items-center text-compact leading-normal text-content-primary transition-colors duration-[var(--motion-fast)] motion-reduce:transition-none", !isSelected && "hover:bg-surface-subtle", isSelected && "selectable-row--selected bg-selection-bg"]}>
    <button {id} type="button" aria-label={ariaLabel} aria-pressed={isSelected} onclick={() => group.select(value)}
        class={["relative flex min-h-16 min-w-0 grow basis-64 items-center gap-3 p-4 text-left cursor-pointer focus-visible:z-10 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-outline-focus [overflow-wrap:anywhere]", inset && "sm:pl-10"]}>
        <span class="min-w-0 grow">
            {@render children()}
        </span>
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
