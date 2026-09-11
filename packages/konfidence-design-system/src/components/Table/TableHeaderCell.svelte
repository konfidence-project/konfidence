<script lang="ts">
    import type { HTMLAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";

    type Sort = "ascending" | "descending" | "none";

    interface Props
        extends Omit<HTMLAttributes<HTMLTableCellElement>, "class" | "children" | "onclick"> {
        class?: string;
        children: Snippet;
        sort?: Sort;
        onsort?: (event: Event) => void;
    }

    let { class: className, children, sort, onsort, ...rest }: Props = $props();

    const cellClass = $derived(
        [
            "sticky top-0 z-[1] whitespace-nowrap text-left align-middle",
            "px-3.5 py-2.5 border-b border-[color:var(--border-subtle)] bg-[color:var(--surface-subtle)]",
            "text-[length:var(--text-meta)] font-semibold uppercase tracking-[0.03em] text-[color:var(--text-tertiary)]",
            className,
        ]
            .filter(Boolean)
            .join(" "),
    );
</script>

<th class={cellClass} aria-sort={sort} {...rest}>
    {#if onsort}
        <button
            type="button"
            class="inline-flex items-center gap-1.5 border-0 bg-transparent p-0 font-inherit text-inherit uppercase tracking-inherit cursor-pointer focus-visible:outline-none focus-visible:text-[color:var(--text-primary)] focus-visible:shadow-[var(--focus-ring)] focus-visible:rounded-[var(--radius-sm)]"
            data-sort-order={sort ?? "none"}
            onclick={onsort}
        >
            {@render children()}
        </button>
    {:else}
        {@render children()}
    {/if}
</th>
