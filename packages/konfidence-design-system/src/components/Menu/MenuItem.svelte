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

    const composedClass = $derived([
        "menu__item",
        variant === "danger" ? "menu__item--danger" : "",
        active ? "menu__item--active" : "",
        className ?? "",
    ].filter(Boolean).join(" "));
</script>

<SkMenu.Item class={composedClass} {...rest} />
