<script lang="ts">
    import type { Component, Snippet } from "svelte";
    import { untrack } from "svelte";
    import type { SessionStore } from "$lib/auth/session.svelte";
    import { provideSession } from "$lib/auth/session.svelte";

    interface Props {
        session: SessionStore;
        component?: Component<Record<string, never>>;
        children?: Snippet;
    }

    let { session, component: Page, children }: Props = $props();

    // The session is captured once at component setup; propagating changes is
    // unnecessary for tests. `untrack` silences the state_referenced_locally
    // warning without altering behaviour.
    untrack(() => provideSession(session));
</script>

{#if Page}
    <Page />
{/if}
{@render children?.()}
