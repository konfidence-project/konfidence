<script lang="ts">
    import { page } from "$app/state";

    import ErrorView from "$lib/components/ErrorView.svelte";
</script>

<ErrorView
    status={page.status}
    error={page.error}
    message="This is the default Error Message"
    actionHref="/"
    actionText="Back to dashboard"
/>
