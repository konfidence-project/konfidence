<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    type SkSeparatorProps = ComponentProps<typeof SkMenu.Separator>;

    let { class: className, ...rest }: SkSeparatorProps = $props();

    const composedClass = $derived(
        className
            ? `menu__sep my-2 h-px bg-[var(--border-subtle)] ${className}`
            : "menu__sep my-2 h-px bg-[var(--border-subtle)]",
    );
</script>

<SkMenu.Separator class={composedClass} {...rest} />
