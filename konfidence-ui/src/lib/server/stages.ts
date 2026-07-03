import { error } from "@sveltejs/kit";

import type { StageList } from "$lib/stages";

export const fetchStages = async () => {
  const response = await fetch("/api/stages");
  console.log("stages.ts: fetching stages");
  if (!response.ok) {
    error(response.status, "Failed to load stages");
  }

  return ((await response.json()) as StageList).items;
};
