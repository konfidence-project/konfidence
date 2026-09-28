<script lang="ts">
    import { Brandbar, Button } from "@konfidence/design-system/components";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import { EMBEDDED_ON, EMBEDDED_QUERY, isEmbedded } from "$lib/shell/embedded";

    const status = $derived(page.status);
    const message = $derived(page.error?.message ?? "Something went wrong.");
    const homeHref = $derived(
        isEmbedded(page.url) ? `${resolve("/")}?${EMBEDDED_QUERY}=${EMBEDDED_ON}` : resolve("/"),
    );
</script>

<svelte:head>
    <title>Konfidence – Error {status}</title>
</svelte:head>

<Brandbar />
<main
    class="flex min-h-[calc(100vh-4px)] flex-col items-center justify-center gap-4 bg-[image:var(--gradient-hero-bg)] p-8 text-center"
>
    <p
        class="m-0 text-content-tertiary font-ui-mono font-ui-bold uppercase tracking-[0.14em] text-ui-sm"
    >
        Error {status}
    </p>
    <h1
        class="m-0 max-w-[40ch] text-content-primary font-ui-display text-ui-h1 [letter-spacing:var(--tracking-h1)]"
    >
        {message}
    </h1>
    <Button variant="primary" href={homeHref} data-testid="error-home">
        Back to dashboard
    </Button>
</main>
