import * as valibot from "valibot";
import { urlTextSchema, validateUrlFilters } from "#lib/url-schemas.ts";
import type { PageLoad } from "./$types";

export const load = (({ params, url }) => {
  const { projectId } = params;
  const filters = validateUrlFilters(
    url,
    valibot.object({
      landscapeId: urlTextSchema,
    }),
  );

  return {
    landscapeId: filters.landscapeId || undefined,
    projectId,
  };
}) satisfies PageLoad;
