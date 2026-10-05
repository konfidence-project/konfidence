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
</script>

<!-- Zag focuses the content for roving tabindex; highlighted items communicate focus. -->
<SkMenu.Content class={[
    "menu min-w-[200px] max-w-[320px] rounded-container bg-surface-default p-2 shadow-elevation-lg focus:outline-none focus-visible:outline-none",
    SIZE_CLASS[size],
    header && "menu--header",
    className,
]} {...rest} />

<style>
    /* applied to a nested <Menu.Header>'s own class, not an element in this template */
    :global(.menu--header) :global(.menu__header) {
        display: flex;
        align-items: center;
        gap: var(--space-3);
        padding: var(--space-3) var(--space-3) var(--space-2);
    }
</style>
