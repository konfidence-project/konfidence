<script lang="ts">
    import type { HTMLAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";

    interface Interactive {
        onselect: () => void;
    }
    interface Static {
        onselect?: undefined;
    }

    type Props = (Interactive | Static) & {
        children: Snippet;
        selected?: boolean;
        class?: string;
    } & Omit<HTMLAttributes<HTMLTableRowElement>, "class" | "children" | "onclick" | "onkeydown">;

    let {
        onselect,
        selected = false,
        class: className,
        children,
        ...rest
    }: Props = $props();

    const interactive = $derived(onselect !== undefined);
    const handleKeydown = (event: KeyboardEvent): void => {
        if (!onselect) {
            return;
        }
        if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            onselect();
        }
    };
</script>

<tr
    class={[
        "transition-colors duration-[var(--motion-fast)] motion-reduce:transition-none",
        interactive && !selected && "hover:bg-surface-subtle",
        interactive && "table-row--interactive cursor-pointer",
        selected && "table-row--selected",
        className,
    ]}
    tabindex={interactive ? 0 : undefined}
    aria-selected={interactive ? selected : undefined}
    onclick={onselect}
    onkeydown={interactive ? handleKeydown : undefined}
    {...rest}
>
    {@render children()}
</tr>

<style>
    .table-row--interactive:focus-visible {
        outline: 2px solid var(--amber-500);
        outline-offset: -2px;
    }

    .table-row--selected {
        background: var(--selection-bg);
    }

    .table-row--selected > :global(td:first-child) {
        position: relative;
    }

    .table-row--selected > :global(td:first-child)::before {
        content: "";
        position: absolute;
        top: 0;
        bottom: 0;
        left: 0;
        width: 3px;
        background: var(--gradient-amber);
        pointer-events: none;
    }
</style>
