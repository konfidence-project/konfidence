<script lang="ts">
    import ErrorView from "$lib/components/ErrorView.svelte";
    import { page } from "$app/state";
</script>

<ErrorView
    title="Failed to load promotions"
    message="The promotions view is currently unavailable. This is a custom error page just for promotions."
    status={page.status}
    error={page.error}
/>
