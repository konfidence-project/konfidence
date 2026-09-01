<script lang="ts">
    import { onMount } from "svelte";
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { LOGIN_PATH, useSession } from "$lib/auth/session.svelte";

    const session = useSession();

    onMount(() => {
        void (async () => {
            if (session.status === "authenticated" || session.status === "idle") {
                await session.signOut();
                return;
            }
            await goto(resolve(LOGIN_PATH));
        })();
    });
</script>

<svelte:head>
    <title>Konfidence – Signing Out</title>
</svelte:head>

<main class="signing-out" role="status" aria-live="polite">
    <span class="spinner" aria-hidden="true"></span>
    <p>Signing out…</p>
</main>

<style>
    /* TODO(#892): migrate scoped CSS to Tailwind per ADR. */
    .signing-out {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: var(--space-3);
        min-height: 100vh;
        color: var(--text-secondary);
    }
    p {
        margin: 0;
        font-family: var(--font-mono);
        font-size: var(--text-sm);
        letter-spacing: 0.08em;
        text-transform: uppercase;
    }
</style>
