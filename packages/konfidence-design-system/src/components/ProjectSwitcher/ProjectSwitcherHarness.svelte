<script lang="ts">
    import ProjectSwitcher from "./ProjectSwitcher.svelte";

    interface Props {
        name?: string;
    }

    let { name = "Konfidence" }: Props = $props();
</script>

<div style="padding: 16px; background: var(--surface-default);">
    <ProjectSwitcher {name} aria-label="Change project" data-testid="project-switch" />
</div>
