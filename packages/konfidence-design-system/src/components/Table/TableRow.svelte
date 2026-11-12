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
        interactive && "cursor-pointer focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[color:var(--amber-500)]",
        selected && "bg-selection-bg [&>td:first-child]:relative [&>td:first-child]:before:content-[''] [&>td:first-child]:before:absolute [&>td:first-child]:before:inset-y-0 [&>td:first-child]:before:left-0 [&>td:first-child]:before:w-[3px] [&>td:first-child]:before:bg-[image:var(--gradient-amber)] [&>td:first-child]:before:pointer-events-none",
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
