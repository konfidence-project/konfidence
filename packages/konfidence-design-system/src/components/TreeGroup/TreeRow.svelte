<script lang="ts">
    import type { Snippet } from "svelte";
    import { getTreeGroup } from "./context.js";

    interface Props {
        /** Unique within the parent TreeGroup. */
        value: string;
        label: string;
        /** Non-interactive content; these snippets render inside a button. */
        metadata?: Snippet;
        trailing?: Snippet;
        disabled?: boolean;
        class?: string;
    }

    let { value, label, metadata, trailing, disabled = false, class: className }: Props = $props();
    const group = getTreeGroup();
    const selected = $derived(group.selectedValue === value);
</script>

<li>
    <button
        type="button"
        class={[
            "tree-row relative flex min-h-16 w-full cursor-pointer items-center gap-3 border-t border-outline-subtle p-4 text-left transition-colors duration-[var(--motion-fast)] ease-[cubic-bezier(var(--ease))] hover:bg-surface-subtle focus-visible:z-10 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--amber-400)] disabled:cursor-not-allowed disabled:opacity-50 motion-reduce:transition-none",
            selected && "is-selected",
            className,
        ]}
        {disabled}
        aria-pressed={selected}
        onclick={() => group.select(value)}
    >
        <span class="tree-row__content min-w-0 flex-1">
            <span class="tree-group__name block text-compact leading-[1.3] font-semibold text-content-primary [overflow-wrap:anywhere]">{label}</span>
            {#if metadata}
                <span class="tree-group__meta mt-1 flex flex-wrap items-center gap-2">{@render metadata()}</span>
            {/if}
        </span>
        {#if trailing}
            <span class="tree-row__end flex shrink-0 items-center gap-2">{@render trailing()}</span>
        {/if}
    </button>
</li>

<style>
    .tree-row.is-selected {
        background: color-mix(in srgb, var(--amber-400) 6%, var(--surface-default));
    }
    .tree-row.is-selected::before {
        content: "";
        position: absolute;
        left: 0;
        /* Include the row's 1px top border in the full-height marker. */
        top: -1px;
        bottom: 0;
        width: 3px;
        border-radius: 0;
        background: var(--gradient-amber);
    }
</style>
