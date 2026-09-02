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

<main
    class="flex min-h-screen flex-col items-center justify-center gap-3 text-[color:var(--text-secondary)]"
    role="status"
    aria-live="polite"
>
    <span
        class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent"
        aria-hidden="true"
    ></span>
    <p
        class="m-0 font-[family-name:var(--font-mono)] uppercase tracking-[0.08em] [font-size:var(--text-sm)]"
    >
        Signing out…
    </p>
</main>
