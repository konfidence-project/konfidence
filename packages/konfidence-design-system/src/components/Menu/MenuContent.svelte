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
