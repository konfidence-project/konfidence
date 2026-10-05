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
        "transition-colors duration-[var(--motion-fast)]",
        "hover:bg-surface-subtle",
        interactive && "cursor-pointer focus-visible:outline-none focus-visible:shadow-[inset_0_0_0_2px_var(--text-link,var(--btn-primary-fg))]",
        selected && "bg-surface-sunken",
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
