<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    type SkContentProps = ComponentProps<typeof SkMenu.Content>;
    type Size = "sm" | "md" | "lg";

    interface KonfidenceContentProps extends Omit<SkContentProps, "class"> {
        /** Width variant. `md` is the default and maps to no modifier. */
        size?: Size;
        /**
         * When `true`, adds `.menu--header` so the first block is styled
         * as a user-info header (avatar + name + mail). Pair with
         * `<Menu.Header>` inside the default slot.
         */
        header?: boolean;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let { size = "md", header = false, class: className, ...rest }: KonfidenceContentProps = $props();

    const SIZE_CLASS: Record<Size, string> = {
        lg: "menu--lg",
        md: "",
        sm: "menu--sm",
    };

    const composedClass = $derived([
        "menu",
        SIZE_CLASS[size],
        header ? "menu--header" : "",
        className ?? "",
    ].filter(Boolean).join(" "));
</script>

<SkMenu.Content class={composedClass} {...rest} />

<style>
    /* applied to Skeleton's <Menu.Content>, not an element in this template */
    :global(.menu) {
        background: var(--surface-default);
        border-radius: var(--radius-lg);
        box-shadow: var(--shadow-lg);
        padding: var(--space-2);
        min-width: 200px;
        max-width: 320px;
    }
    /* Zag focuses the content element for roving tabindex; the browser's default
       outline would draw a black ring around the whole panel. Focus is already
       communicated on the highlighted item via `[data-highlighted]`. */
    :global(.menu):focus,
    :global(.menu):focus-visible {
        outline: none;
    }
    :global(.menu--header) :global(.menu__header) {
        display: flex;
        align-items: center;
        gap: var(--space-3);
        padding: var(--space-3) var(--space-3) var(--space-2);
    }
</style>
