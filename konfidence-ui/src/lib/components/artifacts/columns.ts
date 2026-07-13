import {
  addColumnFilters,
  addSortBy,
  addTableFilter,
  addVirtualScroll,
  matchFilter,
} from "@humanspeak/svelte-headless-table/plugins";
import { type ArtifactSummary } from "$lib/artifacts";
import { type Readable } from "svelte/store";
import { createTable } from "@humanspeak/svelte-headless-table";

const containsFilter = ({
  filterValue,
  value,
}: {
  filterValue: unknown;
  value: unknown;
}): boolean => String(value).toLocaleLowerCase().includes(String(filterValue).toLocaleLowerCase());

const createArtifactTable = (artifacts: Readable<ArtifactSummary[]>) => {
  // Virtualization must wrap the complete filtered and sorted row pipeline.
  // oxlint-disable-next-line eslint/sort-keys
  const table = createTable(artifacts, {
    virtual: addVirtualScroll({ bufferSize: 10, estimatedRowHeight: 48 }),
    filter: addTableFilter(),
    colFilter: addColumnFilters(),
    sort: addSortBy({ disableMultiSort: true }),
  });
  const textPlugins = {
    colFilter: { fn: containsFilter, initialFilterValue: "" },
    sort: {},
  };
  const exactPlugins = {
    colFilter: { fn: matchFilter, initialFilterValue: undefined },
    sort: {},
  };
  const columns = table.createColumns([
    table.column({ accessor: "displayName", header: "Artifact", plugins: textPlugins }),
    table.column({ accessor: "version", header: "Version", plugins: textPlugins }),
    table.column({ accessor: "manifestType", header: "Manifest", plugins: textPlugins }),
    table.column({ accessor: "reuse", header: "Reusable", plugins: exactPlugins }),
    table.column({ accessor: "status", header: "Status", plugins: exactPlugins }),
    table.column({ accessor: "createdAt", header: "Created", plugins: textPlugins }),
  ]);

  return { columns, table };
};

export default createArtifactTable;
