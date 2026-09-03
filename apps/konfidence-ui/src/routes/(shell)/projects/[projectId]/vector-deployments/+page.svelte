<script lang="ts">
    import { page } from "$app/state";

    /**
     * Placeholder Vector Deployments destination — see landscape/+page.svelte.
     */
    const projectId = $derived(page.params.projectId);
</script>

<svelte:head>
    <title>Vector Deployments · Konfidence</title>
</svelte:head>

<section class="mx-auto flex max-w-[70rem] flex-col gap-4 px-6 py-8">
    <h1
        class="m-0 text-[color:var(--text-primary)] font-[weight:var(--weight-display)] [font-size:var(--text-h1)] [letter-spacing:var(--tracking-h1)]"
        data-testid="page-heading"
    >
        Vector Deployments
    </h1>
    <p class="m-0 text-[color:var(--text-secondary)] [font-size:var(--text-body)]">
        Project <code class="font-[family-name:var(--font-mono)]">{projectId}</code>.
    </p>
</section>
