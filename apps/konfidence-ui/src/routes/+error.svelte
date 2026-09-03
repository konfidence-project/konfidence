<script lang="ts">
    import { Brandbar, Button } from "@konfidence/design-system/components";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";

    const status = $derived(page.status);
    const message = $derived(page.error?.message ?? "Something went wrong.");
</script>

<svelte:head>
    <title>Konfidence – Error {status}</title>
</svelte:head>

<Brandbar />
<main
    class="flex min-h-[calc(100vh-4px)] flex-col items-center justify-center gap-4 bg-[image:var(--gradient-hero-bg)] p-8 text-center"
>
    <p
        class="m-0 text-[color:var(--text-tertiary)] font-[family-name:var(--font-mono)] font-[weight:var(--weight-bold)] uppercase tracking-[0.14em] [font-size:var(--text-sm)]"
    >
        Error {status}
    </p>
    <h1
        class="m-0 max-w-[40ch] text-[color:var(--text-primary)] font-[weight:var(--weight-display)] [font-size:var(--text-h1)] [letter-spacing:var(--tracking-h1)]"
    >
        {message}
    </h1>
    <Button variant="primary" href={resolve("/")} data-testid="error-home">
        Back to dashboard
    </Button>
</main>
