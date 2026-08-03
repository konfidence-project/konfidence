<script lang="ts">
    import "@ui5/webcomponents/dist/Button.js";
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

    import { addCustomCSS } from "@ui5/webcomponents-base/dist/Theming.js";
    import type { VectorDeployment } from "$lib/deployments";
    import { t } from "$lib/stores/i18n.svelte";

    type TableRowClickEventDetail =
        import("@ui5/webcomponents/dist/Table.js").TableRowClickEventDetail;
    type SortColumn = "id" | "status" | "version";
    type SortDirection = "None" | "Ascending" | "Descending";

    addCustomCSS("ui5-flexible-column-layout", ".ui5-fcl-root { width: 100%; }");

    const PAGE_SIZE = 50;
    const ROW_HEIGHT = 45;
    const SORT_KEYS = {
        id: (vectorDeployment: VectorDeployment) => vectorDeployment.id,
        status: (vectorDeployment: VectorDeployment) => vectorDeployment.status,
        version: (vectorDeployment: VectorDeployment) => vectorDeployment.version,
    } as const;

    const { vectorDeployments } = $props<{ vectorDeployments: VectorDeployment[] }>();

    let searchTerm = $state("");
    let selectedVectorDeploymentId = $state<string | undefined>();
    let visibleCount = $state(PAGE_SIZE);
    let sortColumn = $state<SortColumn | undefined>();
    let sortDirection = $state<SortDirection>("None");

    const selectedVectorDeployment = $derived(
        vectorDeployments.find(
            (vectorDeployment: VectorDeployment) =>
                vectorDeployment.id === selectedVectorDeploymentId,
        ),
    );
    const layout = $derived.by(() => {
        if (selectedVectorDeployment) {
            return "TwoColumnsMidExpanded";
        }
        return "OneColumn";
    });
    const filteredVectorDeployments = $derived.by(() => {
        const query = searchTerm.trim().toLowerCase();
        if (!query) {
            return vectorDeployments;
        }
        return vectorDeployments.filter((vectorDeployment: VectorDeployment) =>
            [
                vectorDeployment.id,
                vectorDeployment.repository,
                vectorDeployment.component,
                vectorDeployment.version,
                vectorDeployment.landscape,
                vectorDeployment.stage,
                vectorDeployment.status,
            ].some((value) => value.toLowerCase().includes(query)),
        );
    });
    const sortedVectorDeployments = $derived.by(() => {
        if (!sortColumn || sortDirection === "None") {
            return filteredVectorDeployments;
        }
        const getValue = SORT_KEYS[sortColumn];
        let factor = -1;
        if (sortDirection === "Ascending") {
            factor = 1;
        }
        return [...filteredVectorDeployments].toSorted((left, right) =>
            getValue(left).localeCompare(getValue(right)) * factor,
        );
    });
    const visibleVectorDeployments = $derived(sortedVectorDeployments.slice(0, visibleCount));
    const hasMore = $derived(visibleCount < sortedVectorDeployments.length);

    const sortIndicator = (column: SortColumn): SortDirection => {
        if (sortColumn === column) {
            return sortDirection;
        }
        return "None";
    };

    const toggleSort = (column: SortColumn): void => {
        if (sortColumn !== column) {
            sortColumn = column;
            sortDirection = "Ascending";
        } else if (sortDirection === "Ascending") {
            sortDirection = "Descending";
        } else {
            sortColumn = undefined;
            sortDirection = "None";
        }
        visibleCount = PAGE_SIZE;
    };

    const handleSortKeydown = (event: KeyboardEvent, column: SortColumn): void => {
        if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            toggleSort(column);
        }
    };

    const handleSearchInput = (event: Event): void => {
        searchTerm = (event.target as HTMLInputElement | null)?.value ?? "";
        visibleCount = PAGE_SIZE;
    };

    const handleRowClick = (event: CustomEvent<TableRowClickEventDetail>): void => {
        selectedVectorDeploymentId = event.detail.row.getAttribute("row-key") ?? undefined;
    };
</script>

<ui5-flexible-column-layout class="vector-fcl" {layout} accessible-name={t("VECTOR_LIST_ARIA")}>
    <section slot="startColumn" class="vector-pane vector-list">
        <header class="page-head">
            <ui5-title level="H2">{t("VECTOR_TITLE")}</ui5-title>
            <p class="page-sub">
                {t("VECTOR_SUBTITLE", visibleVectorDeployments.length, sortedVectorDeployments.length)}
            </p>
        </header>

        <div class="vt-search" role="search">
            <ui5-input
                class="vt-search-input"
                placeholder={t("VECTOR_SEARCH_PLACEHOLDER")}
                value={searchTerm}
                onui5-input={handleSearchInput}
            ></ui5-input>
            <ui5-button icon="filter" design="Transparent">{t("VECTOR_BTN_FILTER")}</ui5-button>
            <ui5-button icon="filter" design="Transparent">{t("VECTOR_BTN_STATUS")}</ui5-button>
        </div>

        <ui5-table
            class="vector-table"
            overflow-mode="Scroll"
            no-data-text={t("VECTOR_NO_DATA")}
            accessible-name={t("VECTOR_TABLE_ARIA")}
            onui5-row-click={handleRowClick}
        >
            <ui5-table-virtualizer
                slot="features"
                row-count={visibleVectorDeployments.length}
                row-height={ROW_HEIGHT}
            ></ui5-table-virtualizer>
            {#if hasMore}
                <ui5-table-growing slot="features" mode="Scroll" onui5-load-more={() => {
                    visibleCount = Math.min(visibleCount + PAGE_SIZE, sortedVectorDeployments.length);
                }}></ui5-table-growing>
            {/if}

            <ui5-table-header-row slot="headerRow">
                <ui5-table-header-cell
                    class="sortable"
                    role="button"
                    tabindex="0"
                    sort-indicator={sortIndicator("id")}
                    onclick={() => toggleSort("id")}
                    onkeydown={(event: KeyboardEvent) => handleSortKeydown(event, "id")}
                >{t("VECTOR_COL_DEPLOYMENT")}</ui5-table-header-cell>
                <ui5-table-header-cell min-width="220px">{t("VECTOR_COL_REPOSITORY")}</ui5-table-header-cell>
                <ui5-table-header-cell
                    class="sortable"
                    role="button"
                    tabindex="0"
                    min-width="120px"
                    sort-indicator={sortIndicator("version")}
                    onclick={() => toggleSort("version")}
                    onkeydown={(event: KeyboardEvent) => handleSortKeydown(event, "version")}
                >{t("VECTOR_COL_VERSION")}</ui5-table-header-cell>
                <ui5-table-header-cell min-width="140px">{t("VECTOR_COL_LANDSCAPE")}</ui5-table-header-cell>
                <ui5-table-header-cell min-width="180px">{t("VECTOR_COL_STAGE")}</ui5-table-header-cell>
                <ui5-table-header-cell
                    class="sortable"
                    role="button"
                    tabindex="0"
                    min-width="190px"
                    sort-indicator={sortIndicator("status")}
                    onclick={() => toggleSort("status")}
                    onkeydown={(event: KeyboardEvent) => handleSortKeydown(event, "status")}
                >{t("VECTOR_COL_STATUS")}</ui5-table-header-cell>
            </ui5-table-header-row>

            {#each visibleVectorDeployments as vectorDeployment (vectorDeployment.id)}
                <ui5-table-row interactive row-key={vectorDeployment.id}>
                    <ui5-table-cell><span class="vid-cell">{vectorDeployment.id}</span></ui5-table-cell>
                    <ui5-table-cell><span class="mono muted">{vectorDeployment.repository}</span></ui5-table-cell>
                    <ui5-table-cell><span class="mono">{vectorDeployment.version}</span></ui5-table-cell>
                    <ui5-table-cell>{vectorDeployment.landscape}</ui5-table-cell>
                    <ui5-table-cell>{vectorDeployment.stage}</ui5-table-cell>
                    <ui5-table-cell>
                        <ui5-tag design={vectorDeployment.status === "ArtifactDeploymentCreated" ? "Positive" : "Information"}>
                            {vectorDeployment.status}
                        </ui5-tag>
                    </ui5-table-cell>
                </ui5-table-row>
            {/each}
        </ui5-table>
    </section>

    {#if selectedVectorDeployment}
        <section slot="midColumn" class="vector-pane vector-details">
            <header class="details-head">
                <ui5-title level="H3">{selectedVectorDeployment.id}</ui5-title>
                <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-button provides button semantics and keyboard handling) -->
                <ui5-button
                    icon="decline"
                    design="Transparent"
                    accessible-name={t("VECTOR_CLOSE_DETAILS_ARIA")}
                    onclick={() => { selectedVectorDeploymentId = undefined; }}
                ></ui5-button>
            </header>
            <div class="details-body">
                <ui5-illustrated-message
                    name="TntComponents"
                    title-text={t("VECTOR_DETAILS_COMING_SOON_TITLE")}
                    subtitle-text={t("VECTOR_DETAILS_COMING_SOON_SUBTITLE", selectedVectorDeployment.component)}
                >
                    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions (ui5-button provides button semantics and keyboard handling) -->
                    <ui5-button design="Emphasized" onclick={() => { selectedVectorDeploymentId = undefined; }}>
                        {t("VECTOR_BACK_TO_LIST")}
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
    .page-head { display: flex; flex-direction: column; gap: 0.25rem; }
    .page-sub { margin: 0; color: var(--vector-text-muted); font-size: var(--sapFontSize, 0.875rem); }
    .vt-search { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
    .vt-search-input { flex: 1 1 260px; min-width: 200px; }
    .vector-table { flex: 1 1 auto; min-height: 0; }
    .sortable { cursor: pointer; }
    .vid-cell { font-weight: 600; color: var(--vector-text); }
    .mono { font-family: var(--vector-mono-family); font-size: var(--vector-mono-size); color: var(--vector-text); }
    .mono.muted { color: var(--vector-text-muted); }
    .vector-details { gap: 0; padding: 0; }
    .details-head { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 1rem var(--vector-pane-padding); border-bottom: 1px solid var(--sapList_BorderColor, #e4e4e2); }
    .details-body { flex: 1 1 auto; display: flex; align-items: center; justify-content: center; padding: var(--vector-pane-padding); min-height: 0; }
</style>
