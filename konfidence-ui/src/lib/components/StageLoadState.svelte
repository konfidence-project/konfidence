<script lang="ts">
    import "@ui5/webcomponents/dist/BusyIndicator.js";
    import "@ui5/webcomponents/dist/MessageStrip.js";
    import "@ui5/webcomponents/dist/Title.js";

    const { state, error } = $props<{
        state: "loading" | "error";
        error?: unknown;
    }>();

    const message = $derived(
        error instanceof Error ? error.message : "Please try again later.",
    );
</script>

{#if state === "loading"}
    <section class="state-screen" aria-live="polite" aria-labelledby="stages-title">
        <ui5-busy-indicator active size="M"></ui5-busy-indicator>
        <ui5-title level="H2" size="H4">Loading stages</ui5-title>
        <p>Fetching mock Stage resources and their latest conditions.</p>
    </section>
{:else}
    <section class="state-screen" aria-labelledby="stages-title">
        <ui5-message-strip design="Negative" hide-close-button>
            Failed to load stages.
        </ui5-message-strip>
        <p>{message}</p>
    </section>
{/if}

<style>
    .state-screen {
        display: grid;
        place-items: center;
        align-content: center;
        gap: 0.75rem;
        min-height: 18rem;
        padding: 2rem;
        text-align: center;
        box-sizing: border-box;
    }

    .state-screen p {
        max-width: 32rem;
        margin: 0;
        color: var(--sapContent_LabelColor);
    }
</style>
