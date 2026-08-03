<script lang="ts">
    import type { ArtifactDeployment } from "$lib/deployments";
    import { t } from "$lib/stores/i18n.svelte";

    import ArtifactsTable from "./ArtifactsTable.svelte";

    const { deployments }: { deployments: ArtifactDeployment[] } = $props();
</script>

<section class="artifacts" aria-labelledby="artifacts-title">
    <header>
        <h1 id="artifacts-title">{t("ARTIFACTS_TITLE")}</h1>
    </header>

    {#if deployments.length === 0}
        <div class="empty-state">
            <h2>{t("ARTIFACTS_EMPTY")}</h2>
        </div>
    {:else}
        <ArtifactsTable {deployments} />
    {/if}
</section>

<style>
    .artifacts {
        overflow: hidden;
    }

    header,
    .empty-state {
        padding: 1rem;
    }

    h1,
    h2 {
        margin: 0;
    }
</style>
