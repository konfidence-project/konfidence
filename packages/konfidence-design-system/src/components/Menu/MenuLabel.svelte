<script lang="ts">
    import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
    import type { ComponentProps } from "svelte";

    type SkLabelProps = ComponentProps<typeof SkMenu.ItemGroupLabel>;

    let { class: className, ...rest }: SkLabelProps = $props();

</script>

<SkMenu.ItemGroupLabel class={["menu__label pt-1.5 pr-2.5 pb-1 pl-2.5 text-[10px] font-bold tracking-[0.08em] text-content-tertiary uppercase", className]} {...rest} />
