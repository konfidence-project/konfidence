import { getRequestEvent, query } from "$app/server";
import type { ArtifactSummary } from "$lib/artifacts";
import { error } from "@sveltejs/kit";

const getArtifactsData = query(async () => {
  const { fetch } = getRequestEvent();
  const response = await fetch("/api/artifacts");

  if (!response.ok) {
    error(response.status, "Failed to load artifacts");
  }

  return (await response.json()) as {
    apiVersion: "star.konfidence.cloud/v1alpha1";
    items: ArtifactSummary[];
    kind: "ArtifactSummaryList";
  };
});

export { getArtifactsData };
