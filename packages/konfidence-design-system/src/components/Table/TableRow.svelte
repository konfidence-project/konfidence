<script lang="ts">
    import type { HTMLAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";

    type Interactive = {
        onselect: () => void;
    };
    type Static = {
        onselect?: undefined;
    };

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
    const rowClass = $derived(
        [
            "transition-colors duration-[var(--motion-fast)]",
            "hover:bg-[color:var(--surface-subtle)]",
            interactive &&
                "cursor-pointer focus-visible:outline-none focus-visible:shadow-[inset_0_0_0_2px_var(--text-link,var(--btn-primary-fg))]",
            selected && "bg-[color:var(--surface-sunken)]",
            className,
        ]
            .filter(Boolean)
            .join(" "),
    );

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
    class={rowClass}
    tabindex={interactive ? 0 : undefined}
    aria-selected={interactive ? selected : undefined}
    onclick={onselect}
    onkeydown={interactive ? handleKeydown : undefined}
    {...rest}
>
    {@render children()}
</tr>
