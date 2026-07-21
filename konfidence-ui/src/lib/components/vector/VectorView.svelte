<script lang="ts">
    import "@ui5/webcomponents/dist/Button.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents/dist/Input.js";
    import "@ui5/webcomponents/dist/Table.js";
    import "@ui5/webcomponents/dist/TableCell.js";
    import "@ui5/webcomponents/dist/TableGrowing.js";
    import "@ui5/webcomponents/dist/TableHeaderCell.js";
    import "@ui5/webcomponents/dist/TableHeaderRow.js";
    import "@ui5/webcomponents/dist/TableRow.js";
    import "@ui5/webcomponents/dist/TableVirtualizer.js";
    import "@ui5/webcomponents/dist/Tag.js";
    import "@ui5/webcomponents/dist/Title.js";
    import "@ui5/webcomponents-fiori/dist/FlexibleColumnLayout.js";
    import "@ui5/webcomponents-fiori/dist/IllustratedMessage.js";
    import "@ui5/webcomponents-fiori/dist/illustrations/tnt/Components.js";
    import "@ui5/webcomponents-icons/dist/decline.js";
    import "@ui5/webcomponents-icons/dist/filter.js";

    import type { Vector, VectorHealth } from "$lib/vectors.js";
    import { addCustomCSS } from "@ui5/webcomponents-base/dist/Theming.js";

    type TableRowClickEventDetail =
        import("@ui5/webcomponents/dist/Table.js").TableRowClickEventDetail;

    // The FlexibleColumnLayout root does not stretch to fill its host by
    // default, so we make it fill the available width via its shadow DOM.
    addCustomCSS("ui5-flexible-column-layout", ".ui5-fcl-root { width: 100%; }");

    const HEALTH_DESIGN = {
        Healthy: "Positive",
        Warning: "Critical",
    } as const;

    // Sortable columns and how to extract a comparable value from a vector.
    // `health` is ordered by severity (Healthy < Warning < Error).
    const HEALTH_ORDER: Record<VectorHealth, number> = {
        Error: 2,
        Healthy: 0,
        Warning: 1,
    };
    const SORT_KEYS = {
        artifactCount: (vector: Vector) => vector.spec.artifactCount,
        health: (vector: Vector) => HEALTH_ORDER[vector.status.health],
        name: (vector: Vector) => vector.metadata.name,
    } as const;
    type SortColumn = keyof typeof SORT_KEYS;
    type SortDirection = "None" | "Ascending" | "Descending";

    // Number of rows added to the visible window each time the user scrolls to
    // the end of the table, and the initial page size.
    const PAGE_SIZE = 50;
    // Fixed row height (px) required by the virtualizer to compute the scroll
    // area. Keep in sync with the actual rendered row height.
    const ROW_HEIGHT = 45;

    const { vectors } = $props<{ vectors: Vector[] }>();

    let searchTerm = $state("");
    let selectedVectorName = $state<string | undefined>();
    let visibleCount = $state(PAGE_SIZE);
    let sortColumn = $state<SortColumn | undefined>();
    let sortDirection = $state<SortDirection>("None");

    const isDetailsOpen = $derived(selectedVectorName !== undefined);
    const layout = $derived.by(() => {
        if (isDetailsOpen) {
            return "TwoColumnsMidExpanded";
        }
        return "OneColumn";
    });

    const filteredVectors = $derived.by(() => {
        const query = searchTerm.trim().toLowerCase();
        if (!query) {
            return vectors;
        }
        return vectors.filter((vector: Vector) =>
            [
                vector.metadata.name,
                vector.spec.vector,
                vector.spec.hash,
                ...vector.spec.deployedOn,
            ].some((value) => value.toLowerCase().includes(query)),
        );
    });

    const sortedVectors = $derived.by(() => {
        if (!sortColumn || sortDirection === "None") {
            return filteredVectors;
        }
        const getValue = SORT_KEYS[sortColumn];
        let factor = -1;
        if (sortDirection === "Ascending") {
            factor = 1;
        }
        return [...filteredVectors].toSorted((leftVector: Vector, rightVector: Vector) => {
            const left = getValue(leftVector);
            const right = getValue(rightVector);
            if (left < right) {
                return -1 * factor;
            }
            if (left > right) {
                return 1 * factor;
            }
            return 0;
        });
    });

    // Only the rows within the current window are rendered. Growing reveals
    // more, while the virtualizer keeps the painted DOM small.
    const visibleVectors = $derived(sortedVectors.slice(0, visibleCount));
    const hasMore = $derived(visibleCount < sortedVectors.length);

    const sortIndicator = (column: SortColumn): SortDirection => {
        if (sortColumn === column) {
            return sortDirection;
        }
        return "None";
    };

    // Cycles a column through Ascending -> Descending -> None on each click.
    const toggleSort = (column: SortColumn): void => {
        if (sortColumn !== column) {
            sortColumn = column;
            sortDirection = "Ascending";
        } else if (sortDirection === "Ascending") {
            sortDirection = "Descending";
        } else if (sortDirection === "Descending") {
            sortColumn = undefined;
            sortDirection = "None";
        } else {
            sortDirection = "Ascending";
        }
        visibleCount = PAGE_SIZE;
    };

    const handleSortKeydown = (event: KeyboardEvent, column: SortColumn): void => {
        if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            toggleSort(column);
        }
    };


    const selectedVector = $derived(
        vectors.find((vector: Vector) => vector.metadata.name === selectedVectorName),
    );

    const healthDesign = (health: VectorHealth): "Positive" | "Critical" | "Negative" =>
        HEALTH_DESIGN[health as keyof typeof HEALTH_DESIGN] ?? "Negative";

    const handleSearchInput = (event: Event): void => {
        searchTerm = (event.target as HTMLInputElement | null)?.value ?? "";
        // Reset the window whenever the filter changes.
        visibleCount = PAGE_SIZE;
    };

    const handleLoadMore = (): void => {
        visibleCount = Math.min(visibleCount + PAGE_SIZE, sortedVectors.length);
    };

    const handleRowClick = (event: CustomEvent<TableRowClickEventDetail>): void => {
        selectedVectorName = event.detail.row.getAttribute("row-key") ?? undefined;
    };

    const closeDetails = (): void => {
        selectedVectorName = undefined;
    };
</script>

<ui5-flexible-column-layout
    class="vector-fcl"
    layout={layout}
    accessible-name="Vectors"
>
    <section slot="startColumn" class="vector-pane vector-list">
        <header class="page-head">
            <ui5-title level="H2">Vectors</ui5-title>
            <p class="page-sub">
                Immutable, versioned application packages · click a row to view its
                history · showing {visibleVectors.length} of {sortedVectors.length}.
            </p>
        </header>

        <div class="vt-search" role="search">
            <ui5-input
                class="vt-search-input"
                placeholder="Search vectors, artifacts, commits…"
                value={searchTerm}
                onui5-input={handleSearchInput}
            ></ui5-input>
            <ui5-button icon="filter" design="Transparent">Filter</ui5-button>
            <ui5-button icon="filter" design="Transparent">Status</ui5-button>
        </div>

        <ui5-table
            class="vector-table"
            overflow-mode="Scroll"
            no-data-text="No vectors match your search."
            accessible-name="Vectors"
            onui5-row-click={handleRowClick}
        >
            <ui5-table-virtualizer
                slot="features"
                row-count={visibleVectors.length}
                row-height={ROW_HEIGHT}
            ></ui5-table-virtualizer>
            {#if hasMore}
                <ui5-table-growing
                    slot="features"
                    mode="Scroll"
                    onui5-load-more={handleLoadMore}
                ></ui5-table-growing>
            {/if}

            <ui5-table-header-row slot="headerRow">
                <ui5-table-header-cell
                    class="sortable"
                    role="button"
                    tabindex="0"
                    sort-indicator={sortIndicator("name")}
                    onclick={() => toggleSort("name")}
                    onkeydown={(event: KeyboardEvent) => handleSortKeydown(event, "name")}
                >Vector</ui5-table-header-cell>
                <ui5-table-header-cell min-width="120px">Hash</ui5-table-header-cell>
                <ui5-table-header-cell
                    class="sortable"
                    role="button"
                    tabindex="0"
                    min-width="120px"
                    sort-indicator={sortIndicator("artifactCount")}
                    onclick={() => toggleSort("artifactCount")}
                    onkeydown={(event: KeyboardEvent) =>
                        handleSortKeydown(event, "artifactCount")}
                >Artifacts</ui5-table-header-cell>
                <ui5-table-header-cell min-width="360px">Deployed on</ui5-table-header-cell>
                <ui5-table-header-cell
                    class="sortable"
                    role="button"
                    tabindex="0"
                    min-width="140px"
                    sort-indicator={sortIndicator("health")}
                    onclick={() => toggleSort("health")}
                    onkeydown={(event: KeyboardEvent) =>
                        handleSortKeydown(event, "health")}
                >Status</ui5-table-header-cell>
            </ui5-table-header-row>

            {#each visibleVectors as vector (vector.metadata.name)}
                <ui5-table-row interactive row-key={vector.metadata.name}>
                    <ui5-table-cell>
                        <span class="vid-cell">{vector.metadata.name}</span>
                    </ui5-table-cell>
                    <ui5-table-cell>
                        <span class="mono muted">{vector.spec.hash}</span>
                    </ui5-table-cell>
                    <ui5-table-cell>{vector.spec.artifactCount}</ui5-table-cell>
                    <ui5-table-cell>
                        <span class="mono">
                            {vector.spec.deployedOn.join(", ")}
                        </span>
                    </ui5-table-cell>
                    <ui5-table-cell>
                        <ui5-tag design={healthDesign(vector.status.health)}>
                            {vector.status.health.toLowerCase()}
                        </ui5-tag>
                    </ui5-table-cell>
                </ui5-table-row>
            {/each}
        </ui5-table>
    </section>

    {#if isDetailsOpen}
        <section slot="midColumn" class="vector-pane vector-details">
            <header class="details-head">
                <ui5-title level="H3">{selectedVector.metadata.name}</ui5-title>
                <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-button provides button semantics and keyboard handling) -->
                <ui5-button
                    icon="decline"
                    design="Transparent"
                    accessible-name="Close vector details"
                    onclick={closeDetails}
                ></ui5-button>
            </header>
            <div class="details-body">
                <ui5-illustrated-message
                    name="TntComponents"
                    title-text="Vector details coming soon"
                    subtitle-text={`Vector breakdown and artifact informations for ${selectedVector.metadata.name} will appear here.`}
                >
                    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-button provides button semantics and keyboard handling) -->
                    <ui5-button design="Emphasized" onclick={closeDetails}>
                        Back to list
                    </ui5-button>
                </ui5-illustrated-message>
            </div>
        </section>
    {/if}
</ui5-flexible-column-layout>

<style>
    .vector-fcl {
        flex: 1;
        height: 100%;

        --vector-pane-padding: 1rem;
        --vector-text: var(--sapTextColor, #1d2d3e);
        --vector-text-muted: var(--sapContent_LabelColor, #556b82);
        --vector-mono-family: var(--sapFontMonospaceFamily, monospace);
        --vector-mono-size: var(--sapFontSmallSize, 0.75rem);
    }

    .vector-pane {
        display: flex;
        flex-direction: column;
        height: 100%;
        min-height: 0;
        padding: var(--vector-pane-padding);
        gap: 1rem;
        overflow: auto;
        box-sizing: border-box;
    }

    .page-head {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .page-sub {
        margin: 0;
        color: var(--vector-text-muted);
        font-size: var(--sapFontSize, 0.875rem);
    }

    .vt-search {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    .vt-search-input {
        flex: 1 1 260px;
        min-width: 200px;
    }

    .vector-table {
        flex: 1 1 auto;
        min-height: 0;
    }

    .sortable {
        cursor: pointer;
    }

    .vid-cell {
        font-weight: 600;
        color: var(--vector-text);
    }

    .mono {
        font-family: var(--vector-mono-family);
        font-size: var(--vector-mono-size);
        color: var(--vector-text);
    }

    .mono.muted {
        color: var(--vector-text-muted);
    }

    .vector-details {
        gap: 0;
        padding: 0;
    }

    .details-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
        padding: 1rem var(--vector-pane-padding);
        border-bottom: 1px solid var(--sapList_BorderColor, #e4e4e2);
    }

    .details-body {
        flex: 1 1 auto;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: var(--vector-pane-padding);
        min-height: 0;
    }
</style>
