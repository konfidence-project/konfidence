<script lang="ts">
    import "@ui5/webcomponents/dist/Title.js";
    import errorBurstUrl from "$lib/assets/gpt-image-2-kn700szf3pnqy26t64gtqqgz0s89tnyh-removebg-preview.png";
    import { resolve } from "$app/paths";
    import { t } from "$lib/stores/i18n.svelte";

    const {
        title,
        message,
        error,
        status,
    } = $props<{
        title?: string;
        message?: string;
        error?: unknown;
        status?: number;
    }>();

    const displayTitle = $derived(title ?? t("ERROR_DEFAULT_TITLE"));
    const displayMessage = $derived(message ?? t("ERROR_DEFAULT_MESSAGE"));

    const detail = $derived.by(() => {
        if (error instanceof Error) {
            return error.message;
        }

        if (error && typeof error === "object" && "message" in error) {
            return String(error.message);
        }

        return undefined;
    });
</script>

<section class="error-view" aria-labelledby="error-view-title">
    <div class="supernova" aria-hidden="true">
        <span class="aura"></span>
        <span class="orbit orbit-outer"></span>
        <span class="orbit orbit-inner"></span>
        <span class="particle particle-1"></span>
        <span class="particle particle-2"></span>
        <span class="particle particle-3"></span>
        <span class="particle particle-4"></span>
        <span class="particle particle-5"></span>
        <img class="error-art" src={errorBurstUrl} alt="" />
    </div>

    <div class="content-card">
        {#if status}
            <span class="status-code">{t("ERROR_STATUS_LABEL", status)}</span>
        {/if}

        <ui5-title id="error-view-title" level="H1" size="H2">{displayTitle}</ui5-title
        >
        <p class="message">{displayMessage}</p>

        {#if detail}
            <pre class="detail">{detail}</pre>
        {/if}

        <div class="actions">
            <a class="action-link" href={resolve("/")}>{t("ERROR_BACK_TO_START")}</a>
        </div>
    </div>
</section>

<style>
    .error-view {
        display: grid;
        place-items: center;
        align-content: center;
        gap: clamp(0.75rem, 2.5vw, 1.5rem);
        min-height: min(44rem, calc(100vh - 4rem));
        padding: clamp(1.5rem, 6vw, 4rem);
        box-sizing: border-box;
    }

    .supernova {
        position: relative;
        display: grid;
        place-items: center;
        width: min(64vw, 34rem);
        min-width: 16rem;
        aspect-ratio: 1;
        margin-bottom: -1.5rem;
    }

    .aura,
    .orbit,
    .particle {
        position: absolute;
        pointer-events: none;
    }

    .aura {
        width: 68%;
        aspect-ratio: 1;
        border-radius: 999px;
        background:
            radial-gradient(
                circle,
                color-mix(in srgb, #fff3c4 70%, transparent) 0 12%,
                transparent 13%
            ),
            radial-gradient(
                circle,
                color-mix(in srgb, #ffbf2f 34%, transparent) 0 34%,
                transparent 67%
            );
        filter: blur(1.6rem);
        opacity: 0.9;
        animation: supernova-pulse 3.8s ease-in-out infinite;
    }

    .orbit {
        border: 1px solid color-mix(in srgb, #f5a000 36%, transparent);
        border-radius: 999px;
        transform: rotate(-16deg) skew(-8deg);
        animation: orbit-drift 7s ease-in-out infinite;
    }

    .orbit-outer {
        width: 78%;
        height: 44%;
    }

    .orbit-inner {
        width: 52%;
        height: 30%;
        border-color: color-mix(in srgb, #ffcf4d 42%, transparent);
        transform: rotate(22deg) skew(10deg);
        animation-duration: 5.8s;
        animation-direction: reverse;
    }

    .particle {
        width: 0.45rem;
        aspect-ratio: 1;
        border-radius: 999px;
        background: #ffb020;
        box-shadow: 0 0 1rem color-mix(in srgb, #ffb020 70%, transparent);
        animation: particle-drift 4.6s ease-in-out infinite;
    }

    .particle-1 {
        inset: 18% auto auto 18%;
    }
    .particle-2 {
        inset: 31% 12% auto auto;
        width: 0.32rem;
        animation-delay: -1.2s;
    }
    .particle-3 {
        inset: auto 22% 20% auto;
        width: 0.56rem;
        animation-delay: -2.4s;
    }
    .particle-4 {
        inset: auto auto 24% 14%;
        width: 0.28rem;
        animation-delay: -3s;
    }
    .particle-5 {
        inset: 11% auto auto 64%;
        width: 0.24rem;
        animation-delay: -1.8s;
    }

    .error-art {
        position: relative;
        width: 78%;
        height: auto;
        filter: drop-shadow(
                0 0 1.75rem color-mix(in srgb, #ffbf2f 46%, transparent)
            )
            drop-shadow(
                0 1.25rem 2.5rem color-mix(in srgb, #f5a000 22%, transparent)
            );
        animation: art-float 5.4s ease-in-out infinite;
    }

    .content-card {
        display: grid;
        gap: 1rem;
        width: min(100%, 42rem);
        padding: clamp(1.25rem, 4vw, 2.25rem);
        border: 1px solid
            color-mix(in srgb, var(--sapList_BorderColor) 78%, transparent);
        border-radius: 1.5rem;
        background: color-mix(
            in srgb,
            var(--sapGroup_ContentBackground) 88%,
            transparent
        );
        box-shadow: 0 1.25rem 4rem
            color-mix(in srgb, var(--sapContent_ShadowColor) 18%, transparent);
        backdrop-filter: blur(1.25rem);
    }

    .status-code {
        width: max-content;
        padding: 0.25rem 0.625rem;
        border-radius: 999px;
        background: color-mix(in srgb, #ffb020 18%, transparent);
        color: var(--sapCriticalTextColor);
        font-size: var(--sapFontSmallSize);
        font-weight: 700;
        letter-spacing: 0.08em;
        text-transform: uppercase;
    }

    .message {
        margin: 0;
        color: var(--sapContent_LabelColor);
        font-size: 1rem;
        line-height: 1.55;
    }

    .detail {
        max-height: 8rem;
        margin: 0;
        padding: 0.875rem 1rem;
        overflow: auto;
        border: 1px solid var(--sapList_BorderColor);
        border-radius: 0.875rem;
        background: color-mix(
            in srgb,
            var(--sapBackgroundColor) 72%,
            transparent
        );
        color: var(--sapNegativeTextColor);
        font:
            0.875rem/1.45 ui-monospace,
            SFMono-Regular,
            Menlo,
            Monaco,
            Consolas,
            "Liberation Mono",
            monospace;
        white-space: pre-wrap;
    }

    .actions {
        display: flex;
        flex-wrap: wrap;
        gap: 0.75rem;
        align-items: center;
        padding-top: 0.25rem;
    }

    .action-link {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        min-height: 2.5rem;
        padding: 0 1rem;
        border-radius: 0.5rem;
        font: inherit;
        font-weight: 600;
        text-decoration: none;
        cursor: pointer;
    }

    .action-link {
        border: 1px solid var(--sapButton_Emphasized_BorderColor);
        background: var(--sapButton_Emphasized_Background);
        color: var(--sapButton_Emphasized_TextColor);
    }

    .action-link:hover {
        background: var(--sapButton_Emphasized_Hover_Background);
    }

    @keyframes supernova-pulse {
        0%,
        100% {
            scale: 0.95;
            opacity: 0.62;
        }

        48% {
            scale: 1.08;
            opacity: 0.95;
        }
    }

    @keyframes orbit-drift {
        0%,
        100% {
            rotate: 0deg;
            scale: 0.98;
        }

        50% {
            rotate: 7deg;
            scale: 1.03;
        }
    }

    @keyframes particle-drift {
        0%,
        100% {
            translate: 0 0;
            opacity: 0.55;
        }

        50% {
            translate: 0.35rem -0.55rem;
            opacity: 1;
        }
    }

    @keyframes art-float {
        0%,
        100% {
            translate: 0 0;
            scale: 1;
        }

        50% {
            translate: 0 -0.35rem;
            scale: 1.015;
        }
    }

    @media (prefers-reduced-motion: reduce) {
        .aura,
        .orbit,
        .particle,
        .error-art {
            animation: none;
        }
    }

    @media (max-width: 42rem) {
        .error-view {
            padding: 1rem;
        }

        .supernova {
            width: min(78vw, 22rem);
            margin-bottom: -0.75rem;
        }

        .content-card {
            border-radius: 1.125rem;
        }
    }
</style>
