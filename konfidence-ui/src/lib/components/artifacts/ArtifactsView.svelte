<script lang="ts">
    import type { ArtifactSummary } from "$lib/artifacts";

    import ArtifactsTable from "./ArtifactsTable.svelte";

    const { artifacts }: { artifacts: ArtifactSummary[] } = $props();
</script>

<section class="artifacts" aria-labelledby="artifacts-title">
    <header>
        <h1 id="artifacts-title">Artifacts</h1>
    </header>

    {#if artifacts.length === 0}
        <div>
            <h2>No artifacts found</h2>
        </div>
    {:else}
        <ArtifactsTable {artifacts} />
    {/if}
</section>

<style>
    header {
        padding: 1rem;
    }
</style>
