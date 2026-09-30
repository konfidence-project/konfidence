import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params }) => {
  const { projectId, landscapeId, stageId } = params;
  return { landscapeId, projectId, stageId };
};
