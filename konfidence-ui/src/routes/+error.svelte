<script lang="ts">
    import ErrorView from "$lib/components/ErrorView.svelte";
    import { page } from "$app/state";
</script>

<ErrorView
    status={page.status}
    error={page.error}
    message="The dashboard hit an unexpected state while rendering this view. The prototype may still be wired to a missing route, loader, or API response."
/>
