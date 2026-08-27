<script lang="ts">
    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import { page } from "$app/state";
    import Brandbar from "$lib/components/Brandbar.svelte";
    import { session } from "$lib/auth/session.svelte";

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

<div class="login-page">
    <Brandbar />
    <main class="login">
        <img
            class="login__logo login__logo--light"
            src="/logos/logo-light.svg"
            alt="Konfidence"
        />
        <img
            class="login__logo login__logo--dark"
            src="/logos/logo-dark.svg"
            alt="Konfidence"
        />
        <h1 class="sr-only">Sign in to Konfidence</h1>
        <div class="login__eyebrow">Reliable · Reproducible · Sovereign</div>
        <div class="login__sub">Sign in to your delivery workspace.</div>

        {#if errorMessage}
            <p class="login__error" role="alert" data-testid="login-error">{errorMessage}</p>
        {/if}

        <a class="btn btn--primary login__cta" href={loginUrl} rel="external" data-testid="sign-in">
            <span class="ico" aria-hidden="true">
                <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                    <path
                        d="M4.777 0a4.739 4.739 0 0 1 4.779 4.777v1.301l6.224 6.225c.14.14.22.331.22.53v2.417a.75.75 0 0 1-.75.75H13c-.57 0-.752-.576-.964-1H10.75a.75.75 0 0 1-.75-.75v-.75H8.75a.75.75 0 0 1-.75-.75v-.765L7.264 12a.773.773 0 0 1-.764-.75v-1.19l-.56-.56H4.75C2.087 9.5 0 7.454 0 4.777A4.739 4.739 0 0 1 4.777 0Zm0 1.5A3.239 3.239 0 0 0 1.5 4.777C1.5 6.612 2.902 8 4.75 8l1.647.015a.75.75 0 0 1 .383.205l1 1c.14.14.22.331.22.53v.735l.736-.012a.757.757 0 0 1 .764.75V12h1.25a.75.75 0 0 1 .75.75v.75l1.105.008c.503.071.662.598.859.992H14.5v-1.356L8.275 6.919a.75.75 0 0 1-.22-.53V4.777A3.24 3.24 0 0 0 4.778 1.5ZM4.75 3.25c.827 0 1.5.673 1.5 1.5s-.673 1.5-1.5 1.5-1.5-.673-1.5-1.5.673-1.5 1.5-1.5Z"
                    />
                </svg>
            </span>
            <span>Continue with SSO</span>
        </a>
    </main>
    <footer class="login__foot">Part of the Apeiro Reference Architecture</footer>
</div>

<style>
    .login-page {
        min-height: 100vh;
        display: flex;
        flex-direction: column;
        background: var(--gradient-hero-bg);
    }
    .login {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: var(--space-8);
        text-align: center;
    }
    .login__logo {
        height: 52px;
        margin-bottom: var(--space-8);
    }
    .sr-only {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border: 0;
    }
    .login__logo--dark {
        display: none;
    }
    :global([data-theme="dark"]) .login__logo--light {
        display: none;
    }
    :global([data-theme="dark"]) .login__logo--dark {
        display: inline-block;
    }
    .login__eyebrow {
        font-family: var(--font-mono);
        font-size: var(--text-sm);
        font-weight: var(--weight-bold);
        letter-spacing: 0.14em;
        text-transform: uppercase;
        color: var(--text-primary);
        margin-bottom: var(--space-3);
    }
    .login__sub {
        font-size: var(--text-body);
        color: var(--text-secondary);
        margin-bottom: var(--space-8);
    }
    .login__error {
        margin: 0 0 var(--space-6);
        padding: var(--space-3) var(--space-5);
        border-radius: var(--radius-md);
        background: var(--status-error-bg);
        color: var(--status-error-fg);
        border: 1px solid var(--status-error-bg);
        font-size: var(--text-sm);
        max-width: 420px;
    }
    .login__cta {
        min-width: 340px;
        justify-content: center;
        padding: var(--space-4) var(--space-6);
        font-size: var(--text-body);
    }
    .login__hint {
        margin-top: var(--space-5);
        font-size: var(--text-sm);
        color: var(--text-tertiary);
    }
    .login__hint :global(a) {
        color: var(--text-link);
        font-weight: var(--weight-semibold);
        text-decoration: none;
    }
    .login__hint :global(a:hover) {
        text-decoration: underline;
    }
    .login__foot {
        padding: var(--space-6);
        text-align: center;
        font-size: var(--text-meta);
        color: var(--text-tertiary);
    }
</style>
