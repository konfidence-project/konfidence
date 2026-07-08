import { getRequestEvent, query } from "$app/server";
import type { StageList } from "$lib/stages";
import { error } from "@sveltejs/kit";

export const getStages = query(async () => {
  const { fetch } = getRequestEvent();
  const response = await fetch("/api/stages");

  if (!response.ok) {
    error(response.status, "Failed to load stages");
  }

  return (await response.json()) as StageList;
});
