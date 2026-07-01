<script lang="ts">
    import "@ui5/webcomponents/dist/Title.js";
    import "@ui5/webcomponents-fiori/dist/Page.js";

    import StageCard from "$lib/components/StageCard.svelte";
    import StageLoadState from "$lib/components/StageLoadState.svelte";
    import StageStatusPill from "$lib/components/StageStatusPill.svelte";
    import type { Stage } from "$lib/stages.js";

    const { data } = $props<{ data: { stages: Promise<Stage[]> } }>();
</script>

<ui5-page background-design="Solid">
    <header slot="header" class="page-header">
        <div>
            <ui5-title id="stages-title" level="H1" size="H2">Stages</ui5-title>
            <p>Mock Stage resources with vectors and controller conditions.</p>
        </div>
        {#await data.stages then stages}
            <StageStatusPill status="Info" label={`${stages.length} stages`} />
        {/await}
    </header>

    {#await data.stages}
        <StageLoadState state="loading" />
    {:then stages}
        <section class="stage-grid" aria-labelledby="stages-title">
            {#each stages as stage (stage.metadata.name)}
                <StageCard {stage} />
            {/each}
        </section>
    {:catch loadError}
        <StageLoadState state="error" error={loadError} />
    {/await}
</ui5-page>

<style>
    ui5-page {
        height: 100%;
    }

    .page-header {
        display: flex;
        align-items: start;
        justify-content: space-between;
        gap: 1rem;
        padding: 1.5rem clamp(1rem, 3vw, 2rem) 1rem;
        background: var(--sapBackgroundColor);
        border-bottom: 1px solid var(--sapList_BorderColor);
        box-sizing: border-box;
    }

    .page-header p {
        margin: 0.375rem 0 0;
        color: var(--sapContent_LabelColor);
    }

    .stage-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(min(100%, 24rem), 1fr));
        gap: 1rem;
        padding: 1.5rem clamp(1rem, 3vw, 2rem);
        box-sizing: border-box;
    }

    @media (max-width: 40rem) {
        .page-header {
            flex-direction: column;
        }
    }
</style>
