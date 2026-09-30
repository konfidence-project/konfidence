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

<div class="ramp">
    <h3 class="ramp__title">{title}</h3>
    <div class="ramp__grid">
        {#each items as item (item.name)}
            <ColorSwatch name={item.name} value={item.cssVar} label={item.step ?? item.name} />
        {/each}
    </div>
</div>

<style>
    .ramp {
        display: flex;
        flex-direction: column;
        gap: 10px;
        width: 100%;
    }

    .ramp__title {
        margin: 0;
        font-size: var(--text-sm);
        font-weight: var(--weight-semibold, 600);
        color: var(--text-primary);
        text-transform: uppercase;
        letter-spacing: 0.06em;
    }

    .ramp__grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        gap: 12px;
    }
</style>
