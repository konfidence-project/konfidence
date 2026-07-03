<script lang="ts">
    import ErrorView from "$lib/components/ErrorView.svelte";
    import { page } from "$app/state";
</script>

<ErrorView
    status={page.status}
    error={page.error}
    message="This is the default Error Message"
    actionHref="/"
    actionText="Back to dashboard"
/>
