<script lang="ts">
    import { Spinner } from "$lib/components/ui/spinner/index.js";

    const {
        title = "Loading",
        message = "Fetching the requested content.",
        labelledBy,
    } = $props<{ title?: string; message?: string; labelledBy?: string }>();
    const titleId = $derived(labelledBy ?? "loading-view-title");
</script>

<section
    class="grid min-h-72 content-center place-items-center gap-3 p-8 text-center"
    aria-live="polite"
    aria-busy="true"
    aria-labelledby={titleId}
>
    <Spinner class="size-7 text-primary" aria-hidden="true" />
    <h2 class="text-xl font-semibold" id={titleId}>{title}</h2>
    <p class="max-w-lg text-muted-foreground">{message}</p>
</section>
