<script lang="ts">
    import ColorSwatch from "./ColorSwatch.svelte";

    interface Item {
        name: string;
        step?: string;
        cssVar: string;
    }

    interface Props {
        title: string;
        items: readonly Item[];
    }

    let { title, items }: Props = $props();
</script>

<div class="ramp flex w-full flex-col gap-2.5">
    <h3 class="ramp__title m-0 text-compact font-semibold tracking-[0.06em] text-content-primary uppercase">{title}</h3>
    <div class="ramp__grid grid grid-cols-[repeat(auto-fill,minmax(120px,1fr))] gap-3">
        {#each items as item (item.name)}
            <ColorSwatch name={item.name} value={item.cssVar} label={item.step ?? item.name} />
        {/each}
    </div>
</div>
