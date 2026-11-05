<script lang="ts">
    import type { Snippet } from "svelte";
    import { setTreeGroup } from "./context.js";

    interface Props {
        label: string;
        /** Non-interactive content below the label, such as tags or counts. */
        metadata?: Snippet;
        /** TreeRow children. Groups expand independently. */
        children: Snippet;
        open?: boolean;
        /** Single selection within this group; retained when collapsed. */
        selectedValue?: string;
        onOpenChange?: (open: boolean) => void;
        onValueChange?: (value: string) => void;
        class?: string;
    }

    let {
        label,
        metadata,
        children,
        open = $bindable(false),
        selectedValue = $bindable<string | undefined>(undefined),
        onOpenChange,
        onValueChange,
        class: className,
    }: Props = $props();

    const id = $props.id();
    setTreeGroup({
        select(value) {
            if (selectedValue === value) {
                return;
            }
            selectedValue = value;
            onValueChange?.(value);
        },
        get selectedValue() { return selectedValue; },
    });

    const toggle = () => {
        open = !open;
        onOpenChange?.(open);
    };
</script>

<div
    class={["tree-group overflow-hidden rounded-container border border-outline-subtle bg-surface-card text-body leading-[1.5] text-content-primary antialiased", open && "is-open", className]}
    role="group"
    aria-labelledby={`${id}-label`}
>
    <button
        type="button"
        class="tree-group__head flex w-full cursor-pointer items-start gap-2 px-4 py-3 text-left select-none hover:bg-surface-subtle focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--amber-400)]"
        aria-expanded={open}
        aria-controls={`${id}-rows`}
        onclick={toggle}
    >
        <svg
            class={["tree-group__chevron mt-0.5 size-4 shrink-0 text-content-tertiary transition-transform duration-[var(--motion-fast)] ease-[cubic-bezier(var(--ease))] motion-reduce:transition-none", open && "rotate-90"]}
            viewBox="0 0 16 16" fill="none" aria-hidden="true"
        ><path d="M6 4l4 4-4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" /></svg>
        <span class="tree-group__content min-w-0 flex-1">
            <span id={`${id}-label`} class="tree-group__name block text-compact leading-[1.3] font-semibold text-content-primary [overflow-wrap:anywhere]">{label}</span>
            {#if metadata}
                <span class="tree-group__meta mt-1 flex flex-wrap items-center gap-2">{@render metadata()}</span>
            {/if}
        </span>
    </button>
    <ul id={`${id}-rows`} class="tree-group__body m-0 list-none p-0" hidden={!open}>
        {@render children()}
    </ul>
</div>
