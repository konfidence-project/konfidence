<script lang="ts">
    import BoxesIcon from "@lucide/svelte/icons/boxes";
    import NetworkIcon from "@lucide/svelte/icons/network";
    import ProjectSelector from "$lib/components/ProjectSelector.svelte";
    import type { components } from "$lib/konfidence-api/schema";

    type Project = components["schemas"]["ProjectResponse"];
    interface NavItem {
        href: string;
        icon: "landscape" | "vector";
        text: string;
    }

    const {
        currentPath,
        items,
        onNavigate,
        onselect,
        projects,
        projectSelectorClass,
        selectedProjectId,
        selectorId,
    } = $props<{
        currentPath: string;
        items: readonly NavItem[];
        onNavigate?: () => void;
        onselect: (projectId: string) => void;
        projects: Project[];
        projectSelectorClass?: string;
        selectedProjectId?: string;
        selectorId: string;
    }>();
</script>

<ProjectSelector
    class={projectSelectorClass}
    id={selectorId}
    {projects}
    {selectedProjectId}
    {onselect}
/>
<nav class="px-2 py-3" aria-label="Delivery">
    <h2 class="px-3 py-2 text-xs font-semibold tracking-[0.05em] text-muted-foreground uppercase">
        Delivery
    </h2>
    <ul class="grid list-none gap-1 p-0">
        {#each items as item (item.href)}
            <li>
                <a
                    class={[
                        "flex min-h-11 items-center gap-[0.7rem] rounded-[0.55rem] border-l-[3px] border-transparent px-3 py-2.5 text-sm text-sidebar-foreground no-underline hover:bg-sidebar-accent/70 [&_svg]:w-[1.05rem] [&_svg]:shrink-0",
                        currentPath.startsWith(item.href) &&
                            "border-l-sidebar-primary bg-sidebar-accent font-semibold text-sidebar-accent-foreground",
                    ]}
                    href={item.href}
                    aria-current={currentPath.startsWith(item.href) ? "page" : undefined}
                    onclick={onNavigate}
                >
                    {#if item.icon === "landscape"}
                        <NetworkIcon aria-hidden="true" />
                    {:else}
                        <BoxesIcon aria-hidden="true" />
                    {/if}
                    <span>{item.text}</span>
                </a>
            </li>
        {/each}
    </ul>
</nav>
