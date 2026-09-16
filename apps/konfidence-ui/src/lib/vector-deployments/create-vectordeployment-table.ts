import { addSortBy } from "@humanspeak/svelte-headless-table/plugins";
import { createTable } from "@humanspeak/svelte-headless-table";
import type { Readable } from "svelte/store";

import type { VectorDeploymentRow } from "./deployments.js";

const createVectordeploymentTable = (deployments: Readable<VectorDeploymentRow[]>) => {
  const table = createTable(deployments, {
    sort: addSortBy({ disableMultiSort: true }),
  });

  const plugins = { sort: {} };

  const columns = table.createColumns([
    table.column({ accessor: "id", header: "Deployment", plugins }),
    table.column({ accessor: "relatedArtifactDeployments", header: "Artifacts", plugins }),
    table.column({ accessor: "landscape", header: "Landscape", plugins }),
    table.column({ accessor: "stageName", header: "Stage", plugins }),
    table.column({ accessor: "component", header: "Vector", plugins }),
    table.column({ accessor: "version", header: "Version", plugins }),
    table.column({ accessor: "repository", header: "Repository", plugins }),
    table.column({ accessor: "status", header: "Status", plugins }),
  ]);

  return { columns, table };
};

export { createVectordeploymentTable };
