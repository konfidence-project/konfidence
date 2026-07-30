<script lang="ts">
    import ArrowDownIcon from "@lucide/svelte/icons/arrow-down";
    import ArrowUpDownIcon from "@lucide/svelte/icons/arrow-up-down";
    import ArrowUpIcon from "@lucide/svelte/icons/arrow-up";
    import ChevronLeftIcon from "@lucide/svelte/icons/chevron-left";
    import SearchIcon from "@lucide/svelte/icons/search";
    import XIcon from "@lucide/svelte/icons/x";

    import type { VectorDeployment } from "$lib/deployments";
    import { getLocalePreference } from "$lib/locale-preference.svelte";

    type SortColumn = "id" | "status" | "version";
    type SortDirection = "ascending" | "descending" | "none";

    const PAGE_SIZE = 12;
    const SORT_KEYS = {
        id: (deployment: VectorDeployment) => deployment.id,
        status: (deployment: VectorDeployment) => deployment.status,
        version: (deployment: VectorDeployment) => deployment.version,
    } as const;

    const { vectorDeployments } = $props<{ vectorDeployments: VectorDeployment[] }>();
    const locale = getLocalePreference();
    let searchTerm = $state("");
    let selectedId = $state<string>();
    let sortColumn = $state<SortColumn>();
    let sortDirection = $state<SortDirection>("none");

    const selected = $derived(
        vectorDeployments.find((deployment: VectorDeployment) => deployment.id === selectedId),
    );
    const filtered = $derived.by(() => {
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
    const sorted = $derived.by(() => {
        if (!sortColumn || sortDirection === "none") {
            return filtered;
        }
        const value = SORT_KEYS[sortColumn];
        let factor = -1;
        if (sortDirection === "ascending") {
            factor = 1;
        }
        return [...filtered].toSorted((left, right) => value(left).localeCompare(value(right)) * factor);
    });

    const directionFor = (column: SortColumn): SortDirection => {
        if (sortColumn === column) {
            return sortDirection;
        }
        return "none";
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
    };

    const updateSearch = (event: Event): void => {
        searchTerm = (event.currentTarget as HTMLInputElement).value;
    };
</script>

<div class={["grid min-h-0 min-w-0 flex-1 bg-app-bg", selected ? "grid-cols-[minmax(24rem,0.75fr)_minmax(28rem,1.25fr)] max-[68rem]:grid-cols-1" : "grid-cols-1"]}>
    <section class={["flex min-h-0 min-w-0 flex-col gap-4 overflow-hidden px-4 py-[1.2rem] max-[68rem]:px-3 max-[47.999rem]:py-4", selected && "max-[68rem]:hidden"]} aria-labelledby="vector-title">
        <header class="grid gap-[0.2rem]">
            <h1 id="vector-title" class="m-0 text-[1.35rem]">{locale.translate("vector.title")}</h1>
            <p class="m-0 text-[0.84rem] text-app-muted">
                {locale.translate("vector.summary", { sorted: sorted.length, total: sorted.length })}
            </p>
        </header>

        <div class="relative flex items-center" role="search">
            <label for="vector-search" class="sr-only">{locale.translate("vector.search")}</label>
            <SearchIcon class="pointer-events-none absolute left-3 z-1 size-4 text-app-muted" aria-hidden="true" />
            <input
                id="vector-search"
                class="input w-full border-app-border bg-app-card pl-9 text-app-text"
                type="search"
                placeholder={locale.translate("vector.searchPlaceholder")}
                value={searchTerm}
                oninput={updateSearch}
            />
        </div>

        <div class="min-h-48 flex-1 overflow-auto rounded-[0.55rem] border border-app-border bg-app-card">
            <table class="table w-full min-w-256 border-collapse text-[0.8rem] text-app-text [&_th]:border-b [&_th]:border-app-border [&_th]:px-3 [&_th]:py-[0.68rem] [&_th]:text-left [&_th]:whitespace-nowrap [&_td]:border-b [&_td]:border-app-border [&_td]:px-3 [&_td]:py-[0.68rem] [&_td]:text-left [&_td]:whitespace-nowrap" aria-label={locale.translate("vector.table")}>
                <thead class="sticky top-0 z-2 bg-app-bg">
                    <tr>
                        <th class="text-[0.7rem] font-bold tracking-[0.035em] text-app-muted uppercase" scope="col" aria-sort={directionFor("id")}>
                            <button class="-m-[0.4rem] inline-flex cursor-pointer items-center gap-[0.35rem] rounded border-0 bg-transparent p-[0.4rem] text-inherit hover:text-app-accent-strong [&_svg]:size-[0.8rem]" type="button" onclick={() => toggleSort("id")}>
                                {locale.translate("vector.columns.deployment")}
                                {#if directionFor("id") === "ascending"}<ArrowUpIcon aria-hidden="true" />
                                {:else if directionFor("id") === "descending"}<ArrowDownIcon aria-hidden="true" />
                                {:else}<ArrowUpDownIcon aria-hidden="true" />{/if}
                            </button>
                        </th>
                        <th class="text-[0.7rem] font-bold tracking-[0.035em] text-app-muted uppercase" scope="col">{locale.translate("vector.columns.repository")}</th>
                        <th class="text-[0.7rem] font-bold tracking-[0.035em] text-app-muted uppercase" scope="col" aria-sort={directionFor("version")}>
                            <button class="-m-[0.4rem] inline-flex cursor-pointer items-center gap-[0.35rem] rounded border-0 bg-transparent p-[0.4rem] text-inherit hover:text-app-accent-strong [&_svg]:size-[0.8rem]" type="button" onclick={() => toggleSort("version")}>
                                {locale.translate("vector.columns.version")}
                                {#if directionFor("version") === "ascending"}<ArrowUpIcon aria-hidden="true" />
                                {:else if directionFor("version") === "descending"}<ArrowDownIcon aria-hidden="true" />
                                {:else}<ArrowUpDownIcon aria-hidden="true" />{/if}
                            </button>
                        </th>
                        <th class="text-[0.7rem] font-bold tracking-[0.035em] text-app-muted uppercase" scope="col">{locale.translate("vector.columns.landscape")}</th>
                        <th class="text-[0.7rem] font-bold tracking-[0.035em] text-app-muted uppercase" scope="col">{locale.translate("vector.columns.stage")}</th>
                        <th class="text-[0.7rem] font-bold tracking-[0.035em] text-app-muted uppercase" scope="col" aria-sort={directionFor("status")}>
                            <button class="-m-[0.4rem] inline-flex cursor-pointer items-center gap-[0.35rem] rounded border-0 bg-transparent p-[0.4rem] text-inherit hover:text-app-accent-strong [&_svg]:size-[0.8rem]" type="button" onclick={() => toggleSort("status")}>
                                {locale.translate("vector.columns.status")}
                                {#if directionFor("status") === "ascending"}<ArrowUpIcon aria-hidden="true" />
                                {:else if directionFor("status") === "descending"}<ArrowDownIcon aria-hidden="true" />
                                {:else}<ArrowUpDownIcon aria-hidden="true" />{/if}
                            </button>
                        </th>
                    </tr>
                </thead>
                <tbody>
                    {#each sorted as deployment (deployment.id)}
                        <tr class={["hover:bg-app-accent/7", selectedId === deployment.id && "bg-app-accent/7 shadow-[inset_3px_0_var(--app-accent)]"]}>
                            <th scope="row">
                                <button class="deployment-link cursor-pointer border-0 bg-transparent p-0 font-bold text-app-text hover:text-app-accent-strong hover:underline" type="button" onclick={() => (selectedId = deployment.id)}>
                                    {deployment.id}
                                </button>
                            </th>
                            <td><span class="select-all font-mono text-xs text-app-text">{deployment.repository}</span></td>
                            <td><span class="select-all font-mono text-xs text-app-text">{deployment.version}</span></td>
                            <td>{deployment.landscape}</td>
                            <td>{deployment.stage}</td>
                            <td>
                                <span class={["inline-flex items-center gap-[0.35rem] rounded-full px-[0.45rem] py-[0.24rem] text-[0.7rem] font-[650]", deployment.status === "ArtifactDeploymentCreated" ? "bg-app-success/14 text-app-success" : "bg-app-accent/13 text-app-accent-strong"]}>
                                    <i class="size-[0.42rem] rounded-full bg-current" aria-hidden="true"></i>{deployment.status}
                                </span>
                            </td>
                        </tr>
                    {:else}
                        <tr><td class="p-12 text-center text-app-muted" colspan="6">{locale.translate("vector.empty")}</td></tr>
                    {/each}
                </tbody>
            </table>
        </div>
    </section>

    {#if selected}
        <section class="flex min-h-0 min-w-0 flex-col border-l border-app-border bg-app-card max-[68rem]:border-l-0" aria-labelledby="details-title">
            <header class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 border-b border-app-border px-5 py-4 max-[68rem]:grid-cols-[auto_minmax(0,1fr)]">
                <button class="btn-icon hover:preset-tonal hidden max-[68rem]:inline-grid [&_svg]:size-[1.1rem]" type="button" aria-label="Back to vector deployments" onclick={() => (selectedId = undefined)}>
                    <ChevronLeftIcon aria-hidden="true" />
                </button>
                <div class="min-w-0">
                    <span class="text-[0.68rem] font-bold tracking-[0.08em] text-app-muted uppercase">Vector deployment</span>
                    <h2 id="details-title" class="m-0 truncate text-[1.15rem]">{selected.id}</h2>
                </div>
                <button class="btn-icon hover:preset-tonal max-[68rem]:hidden [&_svg]:size-[1.1rem]" type="button" aria-label="Close vector deployment details" onclick={() => (selectedId = undefined)}>
                    <XIcon aria-hidden="true" />
                </button>
            </header>
            <div class="grid flex-1 content-center justify-items-center gap-6 overflow-auto p-[clamp(1rem,4vw,3rem)]">
                <dl class="card m-0 grid w-[min(40rem,100%)] grid-cols-2 overflow-hidden border border-app-border bg-app-bg max-[47.999rem]:grid-cols-1 [&>div]:grid [&>div]:gap-1 [&>div]:border-b [&>div]:border-app-border [&>div]:px-4 [&>div]:py-[0.8rem] [&>div:nth-child(odd)]:border-r [&>div:nth-child(odd)]:border-app-border max-[47.999rem]:[&>div:nth-child(odd)]:border-r-0 [&_dt]:text-[0.68rem] [&_dt]:font-bold [&_dt]:text-app-muted [&_dt]:uppercase [&_dd]:m-0 [&_dd]:[overflow-wrap:anywhere] [&_code]:font-mono [&_code]:text-xs [&_code]:text-app-muted">
                    <div><dt>Component</dt><dd>{selected.component}</dd></div>
                    <div><dt>Version</dt><dd><code>{selected.version}</code></dd></div>
                    <div><dt>Repository</dt><dd><code>{selected.repository}</code></dd></div>
                    <div><dt>Landscape</dt><dd>{selected.landscape}</dd></div>
                    <div><dt>Stage</dt><dd>{selected.stage}</dd></div>
                    <div><dt>Status</dt><dd>{selected.status}</dd></div>
                </dl>
                <div class="grid max-w-lg justify-items-center gap-2 text-center">
                    <span class="mb-1 grid size-16 -rotate-5 place-items-center rounded-[1.2rem] bg-gradient-to-br from-app-accent to-app-warning font-extrabold text-white" aria-hidden="true">KD</span>
                    <h3 class="m-0">Deployment history coming soon</h3>
                    <p class="m-0 text-app-muted">Artifact breakdown and deployment events for {selected.component} will appear here.</p>
                </div>
                <button class="btn preset-filled-primary-500" type="button" onclick={() => (selectedId = undefined)}>Back to list</button>
            </div>
        </section>
    {/if}
</div>
