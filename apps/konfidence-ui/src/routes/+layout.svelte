<script lang="ts">
    import "../theme/app.css";

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
    <!-- The loading indicator is intentionally invisible for the first 200 ms
         so fast identity probes never flash "Loading…". The animation reveals
         it via keyframes defined in src/theme/app.css. -->
    <div
        class="flex min-h-screen items-center justify-center gap-3 text-sm text-[color:var(--text-secondary)] opacity-0 [animation:konfidence-fade-in_240ms_ease_200ms_forwards]"
        role="status"
        aria-live="polite"
    >
        <span
            class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent"
            aria-hidden="true"
        ></span>
        <span class="font-[family-name:var(--font-mono)] uppercase tracking-[0.08em]">
            Loading…
        </span>
    </div>
{/if}
