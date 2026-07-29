<script lang="ts">
    import "@ui5/webcomponents/dist/List.js";
    import "@ui5/webcomponents/dist/ListItemStandard.js";
    import "@ui5/webcomponents/dist/Title.js";
    import "@ui5/webcomponents-fiori/dist/IllustratedMessage.js";
    import "@ui5/webcomponents-fiori/dist/illustrations/tnt/NoApplications.js";

    import { goto } from "$app/navigation";
    import { resolve } from "$app/paths";
    import type { PageProps } from "./$types";

    const { data }: PageProps = $props();
</script>

<section class="projects" aria-labelledby="projects-title">
    <header>
        <ui5-title id="projects-title" level="H1">Projects</ui5-title>
        <p>Select a project to inspect its delivery landscape.</p>
    </header>

    {#if data.projects.length === 0}
        <ui5-illustrated-message
            name="tnt/NoApplications"
            title-text="No projects available"
            subtitle-text="Your account does not currently have access to any projects."
        ></ui5-illustrated-message>
    {:else}
        <ui5-list accessible-name="Projects">
            {#each data.projects as project (project.id)}
                <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-li type Active provides keyboard interaction) -->
                <ui5-li
                    type="Active"
                    description={`Project ID: ${project.id}`}
                    onclick={() =>
                        goto(resolve(`/projects/${project.id}/landscape`))}
                >{project.name}</ui5-li>
            {/each}
        </ui5-list>
    {/if}
</section>

<style>
    .projects {
        display: grid;
        gap: 1.5rem;
        width: min(56rem, 100%);
        margin: 0 auto;
        padding: 2rem;
        box-sizing: border-box;
    }

    header p {
        margin: 0.5rem 0 0;
        color: var(--sapContent_LabelColor);
    }
</style>
