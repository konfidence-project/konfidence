<script lang="ts">
    import { resolve } from "$app/paths";
    import Brandbar from "$lib/components/Brandbar.svelte";
    import { dashboardTitle } from "$lib/dashboard";
    import { useSession } from "$lib/auth/session.svelte";

    const session = useSession();
</script>

<svelte:head>
    <title>{dashboardTitle}</title>
</svelte:head>

<main class="flex min-h-screen flex-col">
    <Brandbar />
    <section
        class="mx-auto flex w-full max-w-[70rem] flex-1 flex-col items-start gap-4 px-12 py-10"
    >
        <h1
            class="m-0 text-[color:var(--text-primary)] font-[weight:var(--weight-display)] [font-size:var(--text-h1)] [letter-spacing:var(--tracking-h1)]"
        >
            {dashboardTitle}
        </h1>
        {#if session.user}
            <p class="m-0 text-[color:var(--text-secondary)] [font-size:var(--text-body)]" data-testid="signed-in-user">
                Signed in as
                <strong class="font-[weight:var(--weight-semibold)] text-[color:var(--text-primary)]">
                    {session.user.name}
                </strong>
            </p>
        {/if}
        <a
            class="btn btn--secondary"
            href={resolve("/logout")}
            data-testid="sign-out"
        >
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
