<script lang="ts">
    import GripIcon from "@lucide/svelte/icons/grip";
    import { Label } from "$lib/components/ui/label/index.js";
    import * as NativeSelect from "$lib/components/ui/native-select/index.js";
    import type { components } from "$lib/konfidence-api/schema";

    type Project = components["schemas"]["ProjectResponse"];

    const {
        class: className,
        collapsed = false,
        id,
        onexpand,
        onselect,
        projects,
        selectedProjectId,
    }: {
        class?: string;
        collapsed?: boolean;
        id: string;
        onexpand?: () => void;
        onselect: (projectId: string) => void;
        projects: Project[];
        selectedProjectId?: string;
    } = $props();

    // Initialize to null because NativeSelect.Root's `ref` is `$bindable(null)`; Svelte would
    // throw `props_invalid_value` if we bound an `undefined` variable to a `null`-defaulted prop.
    // oxlint-disable-next-line unicorn/no-null -- see comment above.
    let selectElement = $state<HTMLSelectElement | null>(null);

    const handleChange = (event: Event): void => {
        const projectId = (event.currentTarget as HTMLSelectElement).value;
        if (projectId) {
            onselect(projectId);
        }
    };

    export const openSelect = (): void => {
        // showPicker is supported by modern browsers; fall back to focus.
        const element = selectElement as
            | (HTMLSelectElement & { showPicker?: () => void })
            | null;
        if (element?.showPicker) {
            element.showPicker();
            return;
        }
        element?.focus();
    };
</script>

<div
    class={[
        "project-switcher grid gap-[0.4rem] border-b border-sidebar-border p-4",
        collapsed &&
            "min-[48rem]:min-h-14 min-[48rem]:place-items-center min-[48rem]:gap-0 min-[48rem]:p-0",
        className,
    ]}
>
    <Label class={[collapsed && "min-[48rem]:hidden"]} for={id}>Project</Label>
    <NativeSelect.Root
        bind:ref={selectElement}
        class={["w-full", collapsed && "min-[48rem]:hidden"]}
        {id}
        aria-label="Project"
        value={selectedProjectId ?? ""}
        onchange={handleChange}
    >
        <NativeSelect.Option value="">Select a project</NativeSelect.Option>
        {#each projects as project (project.id)}
            <NativeSelect.Option value={project.id}>{project.name}</NativeSelect.Option>
        {/each}
    </NativeSelect.Root>
    {#if collapsed}
        <div class="hidden min-[48rem]:relative min-[48rem]:block min-[48rem]:min-h-11 min-[48rem]:w-full">
            <button
                type="button"
                class="group absolute inset-y-0 left-0 right-0 flex min-h-11 items-center justify-center gap-[0.7rem] rounded-[0.4rem] border-0 bg-transparent text-sidebar-foreground no-underline hover:right-auto hover:z-10 hover:w-max hover:bg-sidebar hover:pr-3 hover:pl-4 hover:shadow-md [&_svg]:size-[1.1rem] [&_svg]:shrink-0"
                aria-label="Open project selector"
                onclick={() => onexpand?.()}
            >
                <GripIcon aria-hidden="true" />
                <span class="hidden group-hover:inline">Select a project</span>
            </button>
        </div>
    {/if}
</div>
