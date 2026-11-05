<script lang="ts">
    import type { PageProps } from "./$types";
    import PromotionsPage from "#lib/promotions/PromotionsPage.svelte";

    let { data }: PageProps = $props();
</script>

<svelte:head><title>Promotions · Konfidence</title></svelte:head>

{#key data.projectId}
    <PromotionsPage projectId={data.projectId} />
{/key}
