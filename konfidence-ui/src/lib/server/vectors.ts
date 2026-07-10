import type { VectorList } from "$lib/vectors";
import { error } from "@sveltejs/kit";

const fetchVectors = async () => {
  const response = await fetch("/api/vectors");
  console.log("vectors.ts: fetching vectors");
  if (!response.ok) {
    error(response.status, "Failed to load vectors");
  }

  return ((await response.json()) as VectorList).items;
};

export default fetchVectors;
