<script lang="ts">
    import { page } from "$app/state";

    const returnTo = $derived(page.url.searchParams.get("returnTo") ?? "/landscape");
    const loginUrl = $derived(`/api/login?returnTo=${encodeURIComponent(returnTo)}`);
</script>

<svelte:head>
    <title>Konfidence – Sign In</title>
</svelte:head>

<div class="login-page">
    <div class="login-card">
        <img
            class="logo logo-light"
            src="/assets/logo/full/SVG/400_konfidence_logo_light.svg"
            alt="Konfidence"
        />
        <img
            class="logo logo-dark"
            src="/assets/logo/full/SVG/400_konfidence_logo_dark.svg"
            alt="Konfidence"
        />
        <p class="subtitle">Sign in to your delivery workspace.</p>
        <a href={loginUrl} rel="external" class="sign-in-button">Continue with SSO</a>
    </div>
</div>

<style>
    .login-page {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100vh;
        background: var(--sapBackgroundColor, #f5f6f7);
    }

    .login-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 1.5rem;
        padding: 3rem 4rem;
        background: var(--sapGroup_ContentBackground, #fff);
        border-radius: 1rem;
        box-shadow: 0 2px 16px rgba(0, 0, 0, 0.08);
    }

    .logo {
        width: 280px;
        height: auto;
    }

    .logo-dark {
        display: none;
    }

    :global([data-ui5-theme="konfidence-dark"]) .logo-light {
        display: none;
    }

    :global([data-ui5-theme="konfidence-dark"]) .logo-dark {
        display: block;
    }

    .subtitle {
        margin: 0;
        font-size: 1rem;
        color: var(--sapContent_LabelColor, #6a6d70);
    }

    .sign-in-button {
        display: inline-block;
        margin-top: 1rem;
        padding: 0.75rem 2.5rem;
        background: var(--sapButton_Emphasized_Background, #0a6ed1);
        color: var(--sapButton_Emphasized_TextColor, #fff);
        border: none;
        border-radius: 0.5rem;
        font-size: 1rem;
        font-weight: 600;
        text-decoration: none;
        cursor: pointer;
        transition: background 0.15s;
    }

    .sign-in-button:hover {
        background: var(--sapButton_Emphasized_Hover_Background, #085cad);
    }
</style>
