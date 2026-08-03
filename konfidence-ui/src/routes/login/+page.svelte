<script lang="ts">
    import { page } from "$app/state";

    const returnTo = $derived(page.url.searchParams.get("returnTo") ?? "/landscape");
    const loginUrl = $derived(`/api/login?returnTo=${encodeURIComponent(returnTo)}`);
</script>

<svelte:head>
    <title>Konfidence – Sign In</title>
</svelte:head>

<div class="flex h-screen items-center justify-center bg-app-bg">
    <div class="card flex flex-col items-center gap-6 border border-app-border bg-app-card px-16 py-12 shadow-lg">
        <img
            class="h-auto w-[17.5rem] dark:hidden"
            src="/assets/logo/full/SVG/400_konfidence_logo_light.svg"
            alt="Konfidence"
        />
        <img
            class="hidden h-auto w-[17.5rem] dark:block"
            src="/assets/logo/full/SVG/400_konfidence_logo_dark.svg"
            alt="Konfidence"
        />
        <h1 class="m-0 text-2xl font-semibold text-app-text">Sign in to Konfidence</h1>
        <p class="m-0 text-app-muted">Sign in to your delivery workspace.</p>
        <a href={loginUrl} rel="external" class="btn preset-filled-primary-500 mt-4 px-10 py-3 font-semibold">Continue with SSO</a>
    </div>
</div>
