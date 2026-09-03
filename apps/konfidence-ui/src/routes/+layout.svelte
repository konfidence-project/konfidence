<script lang="ts">
    import "../app.css";

    import type { Snippet } from "svelte";
    import { fade } from "svelte/transition";
    import { OrbitLoader } from "@konfidence/design-system/components";
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import {
        LOGIN_PATH,
        LOGOUT_PATH,
        SessionStore,
        provideSession,
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
         it via the `konfidence-fade-in` keyframes defined in the design
         system's styles/index.css. -->
    <OrbitLoader
        class="min-h-screen items-center justify-center opacity-0 [animation:konfidence-fade-in_240ms_ease_200ms_forwards]"
        label="Loading\u2026"
    />
{/if}
