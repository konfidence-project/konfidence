<script lang="ts">
    import type { Snippet } from "svelte";
    import { fade } from "svelte/transition";
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import {
        LOGIN_PATH,
        LOGOUT_PATH,
        provideSession,
        SessionStore,
    } from "$lib/auth/session.svelte";
    import { getApiClient, setOnUnauthorized } from "$lib/konfidence-api/client-instance";

    interface Props {
        children?: Snippet;
    }

    let { children }: Props = $props();

    const session = provideSession(new SessionStore(getApiClient()));
    setOnUnauthorized(() => session.handleUnauthorized());

    const PUBLIC_ROUTES = new Set<string>([LOGIN_PATH, LOGOUT_PATH]);
    const isPublicRoute = $derived(PUBLIC_ROUTES.has(page.url.pathname));
    const targetReturnTo = $derived(page.url.pathname + page.url.search);
    const showChildren = $derived(isPublicRoute || session.status === "authenticated");

    $effect(() => {
        if (session.status === "idle") {
            void session.refresh();
            return;
        }
        if (!isPublicRoute && session.status === "unauthenticated") {
            const encodedReturnTo = encodeURIComponent(targetReturnTo);
            void goto(resolve(`${LOGIN_PATH}?returnTo=${encodedReturnTo}`));
        }
    });
</script>

{#if showChildren}
    <div in:fade={{ duration: 160 }}>
        {@render children?.()}
    </div>
{:else}
    <div class="loading" role="status" aria-live="polite">
        <span class="spinner" aria-hidden="true"></span>
        <span class="loading__label">Loading…</span>
    </div>
{/if}

<!-- TODO(#892): migrate scoped CSS to Tailwind per ADR. -->
<style>
    .loading {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: var(--space-3);
        min-height: 100vh;
        color: var(--text-secondary);
        font-size: var(--text-sm);
        /* Only reveal the spinner if the identity probe takes noticeable time.
           Avoids a jarring flash of "Loading…" for fast (<200 ms) responses. */
        opacity: 0;
        animation: fade-in var(--motion-base, 240ms) var(--ease, ease) 200ms forwards;
    }
    .loading__label {
        font-family: var(--font-mono);
        letter-spacing: 0.08em;
        text-transform: uppercase;
    }
    @keyframes fade-in {
        to {
            opacity: 1;
        }
    }
</style>
