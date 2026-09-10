<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    type SkSeparatorProps = ComponentProps<typeof SkMenu.Separator>;

    let { class: className, ...rest }: SkSeparatorProps = $props();

    const composedClass = $derived(className ? `menu__sep ${className}` : "menu__sep");
</script>

<SkMenu.Separator class={composedClass} {...rest} />

<style>
    /* applied to Skeleton's <Menu.Separator>, not an element in this template */
    :global(.menu__sep) {
        height: 1px;
        background: var(--border-subtle);
        margin: var(--space-2) 0;
    }
</style>
