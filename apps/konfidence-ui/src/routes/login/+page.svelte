<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import Brandbar from "$lib/components/Brandbar.svelte";
    import { useSession } from "$lib/auth/session.svelte";

    const session = useSession();

    const DEFAULT_RETURN_TO = "/";

    const returnTo = $derived(page.url.searchParams.get("returnTo") ?? DEFAULT_RETURN_TO);
    const errorCode = $derived(page.url.searchParams.get("error"));
    const errorDescription = $derived(page.url.searchParams.get("error_description"));
    const errorMessage = $derived(errorDescription ?? errorCode ?? session.error);
    const loginUrl = $derived(session.buildLoginUrl(returnTo));

    const safeReturnTo = $derived(returnTo.startsWith("/") ? returnTo : DEFAULT_RETURN_TO);

    $effect(() => {
        if (session.status === "idle") {
            void session.refresh();
            return;
        }
        if (session.status === "authenticated") {
            // Client-side navigation for known internal targets; fall back to a
            // hard navigation for arbitrary deep-links the typed resolver
            // cannot statically match.
            if (safeReturnTo === "/") {
                void goto(resolve("/"));
            } else {
                globalThis.location.assign(safeReturnTo);
            }
        }
    });
</script>

<svelte:head>
    <title>Konfidence – Sign In</title>
</svelte:head>

<div class="flex min-h-screen flex-col bg-[image:var(--gradient-hero-bg)]">
    <Brandbar />
    <main class="flex flex-1 flex-col items-center justify-center p-8 text-center">
        <img
            class="mb-8 h-[52px] dark:hidden"
            src="/logos/logo-light.svg"
            alt="Konfidence"
        />
        <img
            class="mb-8 hidden h-[52px] dark:inline-block"
            src="/logos/logo-dark.svg"
            alt="Konfidence"
        />
        <h1 class="sr-only">Sign in to Konfidence</h1>
        <div
            class="mb-3 text-[color:var(--text-primary)] font-[family-name:var(--font-mono)] font-[weight:var(--weight-bold)] uppercase tracking-[0.14em] [font-size:var(--text-sm)]"
        >
            Reliable · Reproducible · Sovereign
        </div>
        <div class="mb-8 text-[color:var(--text-secondary)] [font-size:var(--text-body)]">
            Sign in to your delivery workspace.
        </div>

        {#if errorMessage}
            <p
                class="mb-6 max-w-[420px] rounded-[var(--radius-md)] border border-[color:var(--status-error-bg)] bg-[color:var(--status-error-bg)] px-5 py-3 text-[color:var(--status-error-fg)] [font-size:var(--text-sm)]"
                role="alert"
                data-testid="login-error"
            >
                {errorMessage}
            </p>
        {/if}

        <a
            class="btn btn--primary min-w-[340px] px-6 py-4 [font-size:var(--text-body)]"
            href={loginUrl}
            rel="external"
            data-testid="sign-in"
        >
            <span class="ico" aria-hidden="true">
                <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4">
                    <path
                        d="M4.777 0a4.739 4.739 0 0 1 4.779 4.777v1.301l6.224 6.225c.14.14.22.331.22.53v2.417a.75.75 0 0 1-.75.75H13c-.57 0-.752-.576-.964-1H10.75a.75.75 0 0 1-.75-.75v-.75H8.75a.75.75 0 0 1-.75-.75v-.765L7.264 12a.773.773 0 0 1-.764-.75v-1.19l-.56-.56H4.75C2.087 9.5 0 7.454 0 4.777A4.739 4.739 0 0 1 4.777 0Zm0 1.5A3.239 3.239 0 0 0 1.5 4.777C1.5 6.612 2.902 8 4.75 8l1.647.015a.75.75 0 0 1 .383.205l1 1c.14.14.22.331.22.53v.735l.736-.012a.757.757 0 0 1 .764.75V12h1.25a.75.75 0 0 1 .75.75v.75l1.105.008c.503.071.662.598.859.992H14.5v-1.356L8.275 6.919a.75.75 0 0 1-.22-.53V4.777A3.24 3.24 0 0 0 4.778 1.5ZM4.75 3.25c.827 0 1.5.673 1.5 1.5s-.673 1.5-1.5 1.5-1.5-.673-1.5-1.5.673-1.5 1.5-1.5Z"
                        fill="currentColor"
                    />
                </svg>
            </span>
            <span>Continue with SSO</span>
        </a>
    </main>
    <footer
        class="p-6 text-center text-[color:var(--text-tertiary)] [font-size:var(--text-meta)]"
    >
        Part of the Apeiro Reference Architecture
    </footer>
</div>
