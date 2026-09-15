<script lang="ts">
    import type { HTMLAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";

    interface Props extends Omit<HTMLAttributes<HTMLTableCellElement>, "class" | "children"> {
        class?: string;
        children: Snippet;
    }

    let { class: className, children, ...rest }: Props = $props();

    const cellClass = $derived(
        [
            "px-3.5 py-3 align-middle border-b border-[color:var(--border-subtle)]",
            "[tbody_tr:last-child_&]:border-b-0",
            className,
        ]
            .filter(Boolean)
            .join(" "),
    );
</script>

<td class={cellClass} {...rest}>{@render children()}</td>
