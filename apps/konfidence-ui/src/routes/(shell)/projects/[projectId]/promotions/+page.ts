import * as valibot from "valibot";
import { urlTextSchema, validateUrlFilters } from "#lib/url-schemas.ts";
import type { PageLoad } from "./$types";

export const load = (({ params, url }) => {
  const filters = validateUrlFilters(
    url,
    valibot.object({
      landscapeId: urlTextSchema,
      promotionConfig: urlTextSchema,
      promotionStage: urlTextSchema,
    }),
  );

  return {
    filters: {
      config: filters.promotionConfig,
      landscape: filters.landscapeId,
      stage: filters.promotionStage,
    },
    projectId: params.projectId,
  };
}) satisfies PageLoad;
