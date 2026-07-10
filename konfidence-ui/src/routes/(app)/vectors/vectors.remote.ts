import { getRequestEvent, query } from "$app/server";
import type { VectorList } from "$lib/vectors";
import { error } from "@sveltejs/kit";

// `count` lets callers request a large number of mock vectors to stress-test
// the table's virtualization and growing behavior.
export const getVectors = query("unchecked", async (count?: number) => {
  const { fetch } = getRequestEvent();
  const search = count ? `?count=${count}` : "";
  const response = await fetch(`/api/vectors${search}`);

  if (!response.ok) {
    error(response.status, "Failed to load vectors");
  }

  return (await response.json()) as VectorList;
});
