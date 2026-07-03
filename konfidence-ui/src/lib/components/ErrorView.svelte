<script lang="ts">
    import { resolve } from "$app/paths";
    import "@ui5/webcomponents/dist/MessageStrip.js";
    import "@ui5/webcomponents/dist/Title.js";

    const {
        title = "Something went wrong",
        message = "The requested content could not be loaded.",
        error,
        status,
        actionHref,
        actionText = "Try another page",
    } = $props<{
        title?: string;
        message?: string;
        error?: unknown;
        status?: number;
        actionHref?: ReturnType<typeof resolve>;
        actionText?: string;
    }>();

    const detail = $derived.by(() => {
        if (error instanceof Error) return error.message;

        if (error && typeof error === "object" && "message" in error) {
            return String(error.message);
        }

        return undefined;
    });
</script>

<section class="error-view" aria-labelledby="error-view-title">
    {#if status}
        <span class="status-code">{status}</span>
    {/if}

    <ui5-title id="error-view-title" level="H1" size="H2">{title}</ui5-title>

    <ui5-message-strip design="Negative" hide-close-button>
        {message}
    </ui5-message-strip>

    {#if detail}
        <p>{detail}</p>
    {/if}

    {#if actionHref}
        <a class="action-link" href={resolve(actionHref)}>{actionText}</a>
    {/if}
</section>

<style>
    .error-view {
        display: grid;
        place-items: center;
        align-content: center;
        gap: 1rem;
        min-height: min(32rem, calc(100vh - 9rem));
        padding: 2rem;
        text-align: center;
        box-sizing: border-box;
    }

    .status-code {
        color: var(--sapContent_LabelColor);
        font-size: var(--sapFontHeader6Size);
        font-weight: 700;
        letter-spacing: 0.08em;
        text-transform: uppercase;
    }

    .error-view p {
        max-width: 36rem;
        margin: 0;
        color: var(--sapContent_LabelColor);
    }

    .action-link {
        display: inline-flex;
        align-items: center;
        min-height: 2.25rem;
        padding: 0 1rem;
        border-radius: 0.5rem;
        background: var(--sapButton_Emphasized_Background);
        color: var(--sapButton_Emphasized_TextColor);
        font-weight: 600;
        text-decoration: none;
    }

    .action-link:hover {
        background: var(--sapButton_Emphasized_Hover_Background);
    }
</style>
