import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params }) => ({ projectId: params.projectId });
