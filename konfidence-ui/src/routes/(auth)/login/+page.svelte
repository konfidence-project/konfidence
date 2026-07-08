<script lang="ts">
    import "@ui5/webcomponents/dist/Button.js";
    import "@ui5/webcomponents/dist/Card.js";
    import "@ui5/webcomponents/dist/Title.js";

    import type { IdpType } from "$lib/server/idp-config";
    import { themePreference } from "$lib/stores/theme.svelte";

    const DARK_LOGO_SRC = "/assets/logo/full/SVG/400_konfidence_logo_dark.svg";
    const LIGHT_LOGO_SRC = "/assets/logo/full/SVG/400_konfidence_logo_light.svg";

    const { data } = $props<{
        data: { idpType: IdpType; redirectTo: string };
    }>();

    const logoSrc = $derived.by(() => {
        if (themePreference.selected === "konfidence-dark") {
            return DARK_LOGO_SRC;
        }
        return LIGHT_LOGO_SRC;
    });

    const isStub = $derived(data.idpType === "stub");
    const buttonLabel = $derived.by(() => {
        if (isStub) {
            return "Sign in";
        }
        return "Sign in with SSO";
    });
</script>

<svelte:head>
    <title>Sign in · Konfidence</title>
</svelte:head>

<ui5-card class="login-card" accessible-name="Sign in to Konfidence">
    <div class="login-content">
        <img class="brand-logo" src={logoSrc} alt="Konfidence" />

        <div class="headline">
            <ui5-title level="H1" size="H3">Welcome to Konfidence</ui5-title>
            <p class="subtitle">
                {#if isStub}
                    Real identity providers will hook in here later. For now,
                    just sign in — you'll be assigned a random alias for the
                    session.
                {:else}
                    Continue to your identity provider to sign in.
                {/if}
            </p>
        </div>

        <form method="POST" class="login-form">
            <input
                type="hidden"
                name="redirectTo"
                value={data.redirectTo}
            />
            <ui5-button
                type="Submit"
                design="Emphasized"
                class="login-button"
                >{buttonLabel}</ui5-button
            >
        </form>

        {#if isStub}
            <p class="stub-hint">
                You'll be assigned a randomly generated identity for this
                session.
            </p>
        {/if}
    </div>
</ui5-card>

<style>
    .login-card {
        display: block;
        box-shadow: var(--sapContent_Shadow2);
    }

    .login-content {
        display: grid;
        gap: 1.25rem;
        padding: 2rem clamp(1.25rem, 4vw, 2.25rem);
        text-align: center;
    }

    .brand-logo {
        width: 220px;
        height: auto;
        margin: 0 auto;
        display: block;
    }

    .headline {
        display: grid;
        gap: 0.5rem;
    }

    .subtitle {
        margin: 0;
        color: var(--sapContent_LabelColor);
        font-size: var(--sapFontSize);
        line-height: 1.4;
    }

    .login-form {
        display: flex;
        justify-content: center;
        margin-top: 0.25rem;
    }

    .login-button {
        min-width: 12rem;
    }

    .stub-hint {
        margin: 0;
        color: var(--sapContent_LabelColor);
        font-size: var(--sapFontSmallSize);
    }
</style>
