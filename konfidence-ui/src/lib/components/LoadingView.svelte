<script lang="ts">
    import "@ui5/webcomponents/dist/BusyIndicator.js";
    import "@ui5/webcomponents/dist/Title.js";

    const {
        title = "Loading",
        message = "Fetching the requested content.",
        labelledBy,
    } = $props<{
        title?: string;
        message?: string;
        labelledBy?: string;
    }>();
</script>

<section class="loading-view" aria-live="polite" aria-labelledby={labelledBy}>
    <ui5-busy-indicator active size="M" delay="{0}"></ui5-busy-indicator>
    <ui5-title level="H2" size="H4">{title}</ui5-title>
    <p>{message}</p>
</section>

<style>
    .loading-view {
        display: grid;
        place-items: center;
        align-content: center;
        gap: 0.75rem;
        min-height: 18rem;
        padding: 2rem;
        text-align: center;
        box-sizing: border-box;
    }

    .loading-view p {
        max-width: 32rem;
        margin: 0;
        color: var(--sapContent_LabelColor);
    }
</style>
