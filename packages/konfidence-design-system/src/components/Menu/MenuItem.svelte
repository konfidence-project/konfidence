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

    const BASE_CLASS =
        "menu__item flex w-full items-center gap-2.5 rounded-[var(--radius-sm)] border-none bg-transparent px-2.5 py-2 text-left text-[length:var(--text-sm)] text-[var(--text-primary)] no-underline cursor-pointer hover:bg-[var(--surface-sunken)] focus-visible:bg-[var(--surface-sunken)] focus-visible:outline-none data-highlighted:bg-[var(--surface-sunken)] data-highlighted:outline-none";

    const composedClass = $derived([
        BASE_CLASS,
        variant === "danger" ? "menu__item--danger" : "",
        active ? "menu__item--active" : "",
        className ?? "",
    ].filter(Boolean).join(" "));
</script>

<SkMenu.Item class={composedClass} {...rest} />

<style>
    :global(.menu__item--active) {
        background: var(--amber-50);
        color: var(--amber-800);
        font-weight: var(--weight-semibold);
    }
    :global([data-mode="dark"] .menu__item--active) {
        background: rgba(255, 181, 48, 0.12);
        color: var(--amber-300);
    }
    @media (prefers-color-scheme: dark) {
        :global([data-mode="system"] .menu__item--active) {
            background: rgba(255, 181, 48, 0.12);
            color: var(--amber-300);
        }
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
        font-size: var(--text-meta);
        color: var(--text-tertiary);
        font-weight: var(--weight-regular);
        margin-top: 1px;
    }
</style>
