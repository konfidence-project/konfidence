<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    type SkItemProps = ComponentProps<typeof SkMenu.Item>;
    type Variant = "default" | "danger";

    interface KonfidenceItemProps extends Omit<SkItemProps, "class"> {
        /**
         * `default` (neutral) or `danger` (red, for destructive actions
         * like sign-out or delete).
         */
        variant?: Variant;
        /**
         * Marks the item as the currently-selected option. Sets the
         * `.menu__item--active` amber tint. Used by the project
         * switcher to highlight the current project.
         */
        active?: boolean;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let { variant = "default", active = false, class: className, ...rest }: KonfidenceItemProps = $props();
</script>

<SkMenu.Item class={[
    "menu__item flex w-full items-center gap-2.5 rounded-[var(--radius-sm)] border-none bg-transparent px-2.5 py-2 text-left text-compact text-content-primary no-underline cursor-pointer hover:bg-surface-sunken focus-visible:bg-surface-sunken focus-visible:outline-none data-highlighted:bg-surface-sunken data-highlighted:outline-none",
    variant === "danger" && "menu__item--danger",
    active && "menu__item--active",
    className,
]} {...rest} />

<style>
    :global(.menu__item--active) {
        background: var(--selection-bg);
        color: var(--selection-fg);
        font-weight: var(--weight-semibold);
    }
    :global(.menu__item--danger) {
        color: var(--status-error-fg);
    }
    /* applied by the consumer's item content, not this component's own markup */
    :global(.menu__item) :global(.menu__text) {
        flex: 1;
        min-width: 0;
    }
    :global(.menu__item) :global(.menu__desc) {
        display: block;
        font-size: var(--font-size-meta);
        color: var(--text-tertiary);
        font-weight: var(--weight-regular);
        margin-top: 1px;
    }
</style>
