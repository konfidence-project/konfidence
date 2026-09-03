<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    type SkLabelProps = ComponentProps<typeof SkMenu.ItemGroupLabel>;

    let { class: className, ...rest }: SkLabelProps = $props();

    const composedClass = $derived(className ? `menu__label ${className}` : "menu__label");
</script>

<SkMenu.ItemGroupLabel class={composedClass} {...rest} />
