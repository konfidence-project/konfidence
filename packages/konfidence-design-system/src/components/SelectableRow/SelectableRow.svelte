<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        ariaLabel: string;
        selected?: boolean;
        /** Align desktop content with an expandable header without an outer gutter. */
        inset?: boolean;
        onselect: () => void;
        /** Noninteractive phrasing content; put controls in actions instead. */
        children: Snippet;
        /** Independent controls outside the selection button. */
        actions?: Snippet;
        /** Optional focus-restoration target for the selection button. */
        id?: string;
    }

    let { ariaLabel, selected = false, inset = false, onselect, children, actions, id }: Props = $props();
</script>

<div class={["flex min-w-0 flex-wrap items-center gap-4 px-5 py-4 text-content-primary transition-colors duration-[var(--motion-fast)] hover:bg-surface-subtle motion-reduce:transition-none", inset && "sm:pl-12", selected && "bg-selection-bg"]}>
    <button {id} type="button" aria-label={ariaLabel} aria-pressed={selected} onclick={onselect}
        class="min-w-0 grow basis-64 cursor-pointer rounded-control text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-outline-focus [overflow-wrap:anywhere]">
        {@render children()}
    </button>
    {#if actions}
        <div class="flex min-w-0 flex-wrap items-center gap-4">{@render actions()}</div>
    {/if}
</div>
