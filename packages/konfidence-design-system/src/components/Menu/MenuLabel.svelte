<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    type SkLabelProps = ComponentProps<typeof SkMenu.ItemGroupLabel>;

    let { class: className, ...rest }: SkLabelProps = $props();

    const composedClass = $derived(className ? `menu__label ${className}` : "menu__label");
</script>

<SkMenu.ItemGroupLabel class={composedClass} {...rest} />

<style>
    /* applied to Skeleton's <Menu.ItemGroupLabel>, not an element in this template */
    :global(.menu__label) {
        font-size: 10px;
        font-weight: var(--weight-bold);
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--text-tertiary);
        padding: 6px 10px 4px;
    }
</style>
