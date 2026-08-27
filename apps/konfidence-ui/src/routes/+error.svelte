<script lang="ts">
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import Brandbar from "$lib/components/Brandbar.svelte";

    const status = $derived(page.status);
    const message = $derived(page.error?.message ?? "Something went wrong.");
</script>

<svelte:head>
    <title>Konfidence – Error {status}</title>
</svelte:head>

<Brandbar />
<main class="error-page">
    <p class="error-page__eyebrow">Error {status}</p>
    <h1 class="error-page__title">{message}</h1>
    <a class="btn btn--primary" href={resolve("/")} data-testid="error-home">Back to dashboard</a>
</main>

<style>
    .error-page {
        min-height: calc(100vh - 4px);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: var(--space-4);
        padding: var(--space-8);
        text-align: center;
        background: var(--gradient-hero-bg);
    }
    .error-page__eyebrow {
        margin: 0;
        font-family: var(--font-mono);
        font-size: var(--text-sm);
        font-weight: var(--weight-bold);
        letter-spacing: 0.14em;
        text-transform: uppercase;
        color: var(--text-tertiary);
    }
    .error-page__title {
        margin: 0;
        font-size: var(--text-h1);
        font-weight: var(--weight-display);
        letter-spacing: var(--tracking-h1);
        color: var(--text-primary);
        max-width: 40ch;
    }
</style>
