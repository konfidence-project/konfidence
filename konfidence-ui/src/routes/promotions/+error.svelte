<script lang="ts">
    import { page } from "$app/state";

    import ErrorView from "$lib/components/ErrorView.svelte";
</script>

<ErrorView
    title="Failed to load promotions"
    message="The promotions view is currently unavailable. This is a custom error page just for promotions."
    status={page.status}
    error={page.error}
    actionHref="/landscape"
    actionText="Back to landscape"
/>
