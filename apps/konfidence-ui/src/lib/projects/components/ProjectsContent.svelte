<script lang="ts">
    import type { Snippet } from "svelte";
    import { Button } from "@konfidence/design-system/components";
    import { useProjects } from "$lib/projects/projectContext";

    interface Props { children: Snippet; }
    let { children }: Props = $props();
    const projects = useProjects();
    const errorTitleId = $props.id();
</script>

{#if projects.status === "loading" || projects.status === "idle"}
    <p class="p-6" role="status">Loading projects…</p>
{:else if projects.status === "error"}
    <section class="p-6" aria-labelledby={errorTitleId}>
        <h1 id={errorTitleId}>Projects could not be loaded</h1>
        <p>{projects.error}</p>
        <Button onclick={() => projects.retry()}>Retry</Button>
    </section>
{:else}
    {@render children()}
{/if}
