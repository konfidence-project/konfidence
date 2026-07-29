import {
  addColumnFilters,
  addSortBy,
  addTableFilter,
  addVirtualScroll,
  matchFilter,
} from "@humanspeak/svelte-headless-table/plugins";
import { createTable } from "@humanspeak/svelte-headless-table";
import type { ArtifactDeployment } from "$lib/deployments";
import type { Readable } from "svelte/store";

const containsFilter = ({
  filterValue,
  value,
}: {
  filterValue: unknown;
  value: unknown;
}): boolean => String(value).toLocaleLowerCase().includes(String(filterValue).toLocaleLowerCase());

const createArtifactTable = (deployments: Readable<ArtifactDeployment[]>) => {
  const table = createTable(deployments, {
    colFilter: addColumnFilters(),
    filter: addTableFilter(),
    sort: addSortBy({ disableMultiSort: true }),
    virtual: addVirtualScroll({ bufferSize: 10, estimatedRowHeight: 48 }),
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
    table.column({ accessor: "id", header: "Deployment", plugins: textPlugins }),
    table.column({ accessor: "component", header: "Artifact", plugins: textPlugins }),
    table.column({ accessor: "version", header: "Version", plugins: textPlugins }),
    table.column({ accessor: "repository", header: "Repository", plugins: textPlugins }),
    table.column({ accessor: "landscape", header: "Landscape", plugins: textPlugins }),
    table.column({
      accessor: "vectorDeployments",
      header: "Vector deployments",
      plugins: textPlugins,
    }),
    table.column({ accessor: "stages", header: "Stages", plugins: textPlugins }),
    table.column({ accessor: "status", header: "Status", plugins: exactPlugins }),
  ]);

  return { columns, table };
};

export default createArtifactTable;
