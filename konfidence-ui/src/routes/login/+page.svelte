<script lang="ts">
    import { page } from "$app/state";
    import { buttonVariants } from "$lib/components/ui/button/index.js";

    const returnTo = $derived(page.url.searchParams.get("returnTo") ?? "/landscape");
    const loginUrl = $derived(`/api/login?returnTo=${encodeURIComponent(returnTo)}`);
</script>

<svelte:head>
    <title>Konfidence – Sign In</title>
</svelte:head>

<div class="flex h-screen items-center justify-center bg-background">
    <div class="flex flex-col items-center gap-6 rounded-2xl border bg-card px-16 py-12 shadow-[0_2px_16px_rgb(0_0_0/0.08)]">
        <img
            class="h-auto w-[280px] dark:hidden"
            src="/assets/logo/full/SVG/400_konfidence_logo_light.svg"
            alt="Konfidence"
        />
        <img
            class="hidden h-auto w-[280px] dark:block"
            src="/assets/logo/full/SVG/400_konfidence_logo_dark.svg"
            alt="Konfidence"
        />
        <h1 class="text-2xl font-semibold">Sign in to Konfidence</h1>
        <p class="text-base text-muted-foreground">Sign in to your delivery workspace.</p>
        <a href={loginUrl} rel="external" class={buttonVariants({ class: "mt-4 px-10 py-6 text-base" })}>
            Continue with SSO
        </a>
    </div>
</div>
