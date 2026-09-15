import { addSortBy } from "@humanspeak/svelte-headless-table/plugins";
import { createTable } from "@humanspeak/svelte-headless-table";
import type { Readable } from "svelte/store";

import type { ArtifactDeploymentRow } from "./deployments.js";

const createArtifactTable = (deployments: Readable<ArtifactDeploymentRow[]>) => {
  const table = createTable(deployments, {
    sort: addSortBy({ disableMultiSort: true }),
  });

  const plugins = { sort: {} };

  const columns = table.createColumns([
    table.column({ accessor: "id", header: "Deployment", plugins }),
    table.column({ accessor: "component", header: "Artifact", plugins }),
    table.column({ accessor: "version", header: "Version", plugins }),
    table.column({ accessor: "repository", header: "Repository", plugins }),
    table.column({ accessor: "landscape", header: "Landscape", plugins }),
    table.column({
      accessor: (row) => row.vectorDeploymentLabels.join(", "),
      header: "Vector deployments",
      id: "vectorDeployments",
      plugins,
    }),
    table.column({
      accessor: (row) => row.stageNames.join(", "),
      header: "Stages",
      id: "stages",
      plugins,
    }),
    table.column({ accessor: "status", header: "Status", plugins }),
  ]);

  return { columns, table };
};

export { createArtifactTable };
