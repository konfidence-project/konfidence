<script lang="ts">
    import { page } from "$app/state";
    import ErrorView from "$lib/components/ErrorView.svelte";
    import { HTTP_FORBIDDEN } from "$lib/http-status";

    const title = $derived.by(() => {
        if (page.status === HTTP_FORBIDDEN) {
            return "Access denied";
        }
        return "Application unavailable";
    });
    const message = $derived.by(() => {
        if (page.status === HTTP_FORBIDDEN) {
            return "You are signed in, but your account does not have permission to access this resource.";
        }
        return "The Konfidence API is currently unavailable.";
    });
</script>

<ErrorView {title} {message} status={page.status} error={page.error} />
