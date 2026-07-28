<script lang="ts">
    import ArrowDownIcon from "@lucide/svelte/icons/arrow-down";
    import ArrowUpDownIcon from "@lucide/svelte/icons/arrow-up-down";
    import ArrowUpIcon from "@lucide/svelte/icons/arrow-up";
    import BoxesIcon from "@lucide/svelte/icons/boxes";
    import SearchIcon from "@lucide/svelte/icons/search";
    import XIcon from "@lucide/svelte/icons/x";
    import { Badge } from "$lib/components/ui/badge/index.js";
    import { Button } from "$lib/components/ui/button/index.js";
    import * as Empty from "$lib/components/ui/empty/index.js";
    import { Input } from "$lib/components/ui/input/index.js";
    import * as Sheet from "$lib/components/ui/sheet/index.js";
    import * as Table from "$lib/components/ui/table/index.js";
    import type { VectorDeployment } from "$lib/deployments";
    import { MediaQuery } from "svelte/reactivity";

    type SortColumn = "id" | "status" | "version";
    type SortDirection = "ascending" | "descending" | "none";

    const PAGE_SIZE = 10;
    const SORT_KEYS = {
        id: (deployment: VectorDeployment) => deployment.id,
        status: (deployment: VectorDeployment) => deployment.status,
        version: (deployment: VectorDeployment) => deployment.version,
    } as const;

    const { vectorDeployments } = $props<{ vectorDeployments: VectorDeployment[] }>();
    let searchTerm = $state("");
    let selectedId = $state<string>();
    let visibleCount = $state(PAGE_SIZE);
    let sortColumn = $state<SortColumn>();
    let sortDirection = $state<SortDirection>("none");
    let detailsSheetOpen = $state(false);
    const mobileViewport = new MediaQuery("(max-width: 48rem)", false);

    const selectedDeployment = $derived(
        vectorDeployments.find((item: VectorDeployment) => item.id === selectedId),
    );
    const filteredDeployments = $derived.by(() => {
        const query = searchTerm.trim().toLowerCase();
        if (!query) {
            return vectorDeployments;
        }
        return vectorDeployments.filter((deployment: VectorDeployment) =>
            [
                deployment.id,
                deployment.repository,
                deployment.component,
                deployment.version,
                deployment.landscape,
                deployment.stage,
                deployment.status,
            ].some((value) => value.toLowerCase().includes(query)),
        );
    });
    const sortedDeployments = $derived.by(() => {
        if (!sortColumn || sortDirection === "none") {
            return filteredDeployments;
        }
        const getValue = SORT_KEYS[sortColumn];
        let factor = -1;
        if (sortDirection === "ascending") {
            factor = 1;
        }
        return filteredDeployments.toSorted(
            (left: VectorDeployment, right: VectorDeployment) =>
                getValue(left).localeCompare(getValue(right)) * factor,
        );
    });
    const visibleDeployments = $derived(sortedDeployments.slice(0, visibleCount));
    const hasMore = $derived(visibleCount < sortedDeployments.length);

    const selectDeployment = (id: string): void => {
        selectedId = id;
        detailsSheetOpen = mobileViewport.current;
    };

    const clearSelection = (): void => {
        selectedId = undefined;
        detailsSheetOpen = false;
    };

    const toggleSort = (column: SortColumn): void => {
        if (sortColumn !== column) {
            sortColumn = column;
            sortDirection = "ascending";
        } else if (sortDirection === "ascending") {
            sortDirection = "descending";
        } else {
            sortColumn = undefined;
            sortDirection = "none";
        }
        visibleCount = PAGE_SIZE;
    };

    const ariaSort = (column: SortColumn): SortDirection => {
        if (sortColumn === column) {
            return sortDirection;
        }
        return "none";
    };

    const search = (event: Event): void => {
        searchTerm = (event.currentTarget as HTMLInputElement).value;
        visibleCount = PAGE_SIZE;
    };
</script>

{#snippet sortIcon(column: SortColumn)}
    {#if ariaSort(column) === "ascending"}
        <ArrowUpIcon aria-hidden="true" />
    {:else if ariaSort(column) === "descending"}
        <ArrowDownIcon aria-hidden="true" />
    {:else}
        <ArrowUpDownIcon aria-hidden="true" />
    {/if}
{/snippet}

{#snippet details(deployment: VectorDeployment)}
    <div class="grid gap-5 px-6 pt-16 pb-8">
        <div class="grid size-16 place-items-center rounded-[1.25rem] bg-accent text-primary" aria-hidden="true">
            <BoxesIcon class="w-8" />
        </div>
        <div>
            <span class="text-[0.68rem] font-bold tracking-[0.1em] text-primary uppercase">
                Vector deployment
            </span>
            <h2 class="text-xl font-semibold">{deployment.id}</h2>
            <p class="text-muted-foreground">{deployment.component} is assigned to {deployment.stage}.</p>
        </div>
        <dl class="grid border-t">
            <div class="grid grid-cols-[7rem_minmax(0,1fr)] gap-3 border-b py-3">
                <dt class="text-[0.78rem] text-muted-foreground">Status</dt>
                <dd class="m-0 min-w-0 [overflow-wrap:anywhere]">{deployment.status}</dd>
            </div>
            <div class="grid grid-cols-[7rem_minmax(0,1fr)] gap-3 border-b py-3">
                <dt class="text-[0.78rem] text-muted-foreground">Version</dt>
                <dd class="m-0 min-w-0 [overflow-wrap:anywhere]"><code>{deployment.version}</code></dd>
            </div>
            <div class="grid grid-cols-[7rem_minmax(0,1fr)] gap-3 border-b py-3">
                <dt class="text-[0.78rem] text-muted-foreground">Repository</dt>
                <dd class="m-0 min-w-0 [overflow-wrap:anywhere]"><code>{deployment.repository}</code></dd>
            </div>
            <div class="grid grid-cols-[7rem_minmax(0,1fr)] gap-3 border-b py-3">
                <dt class="text-[0.78rem] text-muted-foreground">Landscape</dt>
                <dd class="m-0 min-w-0 [overflow-wrap:anywhere]">{deployment.landscape}</dd>
            </div>
            <div class="grid grid-cols-[7rem_minmax(0,1fr)] gap-3 border-b py-3">
                <dt class="text-[0.78rem] text-muted-foreground">Stage</dt>
                <dd class="m-0 min-w-0 [overflow-wrap:anywhere]">{deployment.stage}</dd>
            </div>
        </dl>
        <Button variant="outline" onclick={clearSelection}>Back to list</Button>
    </div>
{/snippet}

<div
    class={[
        "grid min-h-full grid-cols-[minmax(0,1fr)]",
        selectedDeployment &&
            "grid-cols-[minmax(36rem,1fr)_minmax(20rem,28rem)] max-[48rem]:block",
    ]}
>
    <section class="min-w-0 p-[clamp(1rem,3vw,2rem)] max-[48rem]:p-4" aria-labelledby="vector-title">
        <header class="mb-4 grid gap-1">
            <span class="text-[0.68rem] font-bold tracking-[0.1em] text-primary uppercase">
                Project inventory
            </span>
            <h1 class="text-[1.55rem] font-semibold tracking-[-0.025em]" id="vector-title">
                Vector Deployments
            </h1>
            <p class="text-muted-foreground">
                Versioned vectors currently assigned to project stages. Showing
                {visibleDeployments.length} of {sortedDeployments.length}.
            </p>
        </header>

        <div class="relative mb-4 max-w-2xl" role="search">
            <SearchIcon
                class="absolute top-1/2 left-[0.7rem] z-1 w-4 -translate-y-1/2 text-muted-foreground"
                aria-hidden="true"
            />
            <Input
                class="pl-[2.15rem]"
                type="search"
                aria-label="Search vector deployments"
                placeholder="Search vector deployments..."
                value={searchTerm}
                oninput={search}
            />
        </div>

        {#if sortedDeployments.length === 0}
            <Empty.Root class="border bg-card">
                <Empty.Media variant="icon"><SearchIcon /></Empty.Media>
                <Empty.Header>
                    <Empty.Title>No matching vector deployments</Empty.Title>
                    <Empty.Description>Try a different name, version, stage, or status.</Empty.Description>
                </Empty.Header>
            </Empty.Root>
        {:else}
            <div class="overflow-hidden rounded-xl border bg-card">
                <Table.Root class="[&_td]:max-[48rem]:px-[0.45rem] [&_th]:max-[48rem]:px-[0.45rem]">
                    <Table.Caption class="sr-only">Vector deployments</Table.Caption>
                    <Table.Header>
                        <Table.Row>
                            <Table.Head aria-sort={ariaSort("id")}>
                                <Button class="-ml-3 font-semibold" variant="ghost" onclick={() => toggleSort("id")}>
                                    Vector deployment {@render sortIcon("id")}
                                </Button>
                            </Table.Head>
                            <Table.Head>Repository</Table.Head>
                            <Table.Head aria-sort={ariaSort("version")}>
                                <Button class="-ml-3 font-semibold" variant="ghost" onclick={() => toggleSort("version")}>
                                    Version {@render sortIcon("version")}
                                </Button>
                            </Table.Head>
                            <Table.Head class="max-[70rem]:hidden">Landscape</Table.Head>
                            <Table.Head class="max-[70rem]:hidden">Stage</Table.Head>
                            <Table.Head class="max-[48rem]:hidden" aria-sort={ariaSort("status")}>
                                <Button class="-ml-3 font-semibold" variant="ghost" onclick={() => toggleSort("status")}>
                                    Status {@render sortIcon("status")}
                                </Button>
                            </Table.Head>
                        </Table.Row>
                    </Table.Header>
                    <Table.Body>
                        {#each visibleDeployments as deployment (deployment.id)}
                            <Table.Row data-state={deployment.id === selectedId ? "selected" : undefined}>
                                <Table.Cell>
                                    <Button
                                        class="h-auto max-w-52 justify-start p-0 text-left font-semibold whitespace-normal text-foreground max-[48rem]:max-w-30 max-[48rem]:[overflow-wrap:anywhere]"
                                        variant="link"
                                        onclick={() => selectDeployment(deployment.id)}
                                    >
                                        {deployment.id}
                                    </Button>
                                </Table.Cell>
                                <Table.Cell>
                                    <code class="block max-w-36 truncate text-xs text-muted-foreground max-[48rem]:max-w-36">
                                        {deployment.repository}
                                    </code>
                                </Table.Cell>
                                <Table.Cell><code class="text-xs">{deployment.version}</code></Table.Cell>
                                <Table.Cell class="max-[70rem]:hidden">{deployment.landscape}</Table.Cell>
                                <Table.Cell class="max-[70rem]:hidden">{deployment.stage}</Table.Cell>
                                <Table.Cell class="max-[48rem]:hidden">
                                    <Badge
                                        variant="outline"
                                        class={deployment.status === "ArtifactDeploymentCreated"
                                            ? "border-success/55 bg-success-background text-success"
                                            : undefined}
                                    >
                                        {deployment.status}
                                    </Badge>
                                </Table.Cell>
                            </Table.Row>
                        {/each}
                    </Table.Body>
                </Table.Root>
            </div>
            {#if hasMore}
                <div class="flex justify-center p-4">
                    <Button
                        variant="outline"
                        onclick={() =>
                            (visibleCount = Math.min(
                                visibleCount + PAGE_SIZE,
                                sortedDeployments.length,
                            ))}
                    >
                        Load more
                    </Button>
                </div>
            {/if}
        {/if}
    </section>

    {#if selectedDeployment}
        <aside
            class="relative border-l bg-card max-[48rem]:hidden"
            aria-label={`Details for ${selectedDeployment.id}`}
        >
            <Button
                class="absolute top-4 right-4 z-1"
                variant="ghost"
                size="icon"
                aria-label="Close vector deployment details"
                onclick={clearSelection}
            >
                <XIcon />
            </Button>
            {@render details(selectedDeployment)}
        </aside>
    {/if}
</div>

<Sheet.Root bind:open={detailsSheetOpen}>
    <Sheet.Content side="right" class="!w-[94vw] !max-w-[34rem] overflow-y-auto">
        {#if selectedDeployment}
            <Sheet.Header class="sr-only">
                <Sheet.Title>{selectedDeployment.id}</Sheet.Title>
                <Sheet.Description>Vector deployment details</Sheet.Description>
            </Sheet.Header>
            {@render details(selectedDeployment)}
        {/if}
    </Sheet.Content>
</Sheet.Root>
