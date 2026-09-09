<script lang="ts">
    import { onMount } from "svelte";
    import { OrbitLoader } from "@konfidence/design-system/components";
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

<main class="flex min-h-screen items-center justify-center">
    <OrbitLoader label="Signing out\u2026" />
</main>
