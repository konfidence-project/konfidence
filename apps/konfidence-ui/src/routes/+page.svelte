<script lang="ts">
    import { resolve } from "$app/paths";
    import Brandbar from "$lib/components/Brandbar.svelte";
    import { dashboardTitle } from "$lib/dashboard";
    import { session } from "$lib/auth/session.svelte";
</script>

<svelte:head>
    <title>{dashboardTitle}</title>
</svelte:head>

<main class="dashboard">
    <Brandbar />
    <section class="dashboard__content">
        <h1>{dashboardTitle}</h1>
        {#if session.user}
            <p class="dashboard__signed-in" data-testid="signed-in-user">
                Signed in as <strong>{session.user.name}</strong>
            </p>
        {/if}
        <a class="btn btn--secondary" href={resolve("/logout")} data-testid="sign-out">
            Sign out
        </a>
        <!-- rel="external" bypasses SvelteKit's client router, so the lint
             rule and the typed `resolve()` are both skipped. The fallback
             serves the SPA shell for /does-not-exist and +error.svelte
             renders because no route matches. -->
        <a
            class="btn btn--secondary"
            href="/does-not-exist"
            rel="external"
            data-testid="show-error-page"
        >
            Show error page
        </a>
    </section>
</main>

<style>
    .dashboard {
        min-height: 100vh;
        display: flex;
        flex-direction: column;
    }
    .dashboard__content {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: var(--space-4);
        padding: var(--space-10) var(--space-12);
        max-width: 1120px;
        margin: 0 auto;
        width: 100%;
    }
    h1 {
        margin: 0;
        font-size: var(--text-h1);
        font-weight: var(--weight-display);
        letter-spacing: var(--tracking-h1);
        color: var(--text-primary);
    }
    .dashboard__signed-in {
        margin: 0;
        color: var(--text-secondary);
        font-size: var(--text-body);
    }
    .dashboard__signed-in strong {
        color: var(--text-primary);
        font-weight: var(--weight-semibold);
    }
</style>
