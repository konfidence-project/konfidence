<script lang="ts">
    import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

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

<section class="grid min-h-72 w-full place-items-center content-center gap-3 p-8 text-center" aria-live="polite" aria-busy="true" aria-labelledby={labelledBy}>
    <LoaderCircleIcon class="size-8 animate-spin text-app-accent" aria-hidden="true" />
    <h2 class="m-0">{title}</h2>
    <p class="m-0 max-w-lg text-app-muted">{message}</p>
</section>
