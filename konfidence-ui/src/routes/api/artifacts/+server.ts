import type { ArtifactSummary } from "$lib/artifacts";
import type { RequestHandler } from "./$types";
import { json } from "@sveltejs/kit";

const DEFAULT_COUNT = 10_000;
const ISO_DATE_LENGTH = 10;
const MINUTES_BETWEEN_ARTIFACTS = 5;
const MAX_COUNT = 100_000;
const MILLISECONDS_PER_MINUTE = 60_000;
const MIN_RESPONSE_DELAY = 100;
const RANDOM_RESPONSE_DELAY = 500;
const REUSE_INTERVAL = 2;
const STATUS_INTERVAL = 3;

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const timestamp = (minutesAgo: number) =>
  new Date(Date.now() - minutesAgo * MILLISECONDS_PER_MINUTE).toISOString();

const alternatingValue = <Value>(index: number, first: Value, second: Value): Value => {
  if (index % REUSE_INTERVAL === 0) {
    return first;
  }
  return second;
};

const status = (index: number): string => {
  if (index % STATUS_INTERVAL === 0) {
    return "Not ready";
  }
  return "Ready";
};

const generateArtifactDeployments = (amount: number): ArtifactSummary[] =>
  Array.from({ length: amount }, (_value, index) => {
    const name = `artifact-${index + 1}`;
    const version = `1.0.${index}`;

    return {
      createdAt: timestamp(index * MINUTES_BETWEEN_ARTIFACTS).slice(0, ISO_DATE_LENGTH),
      displayName: name,
      manifestType: alternatingValue(index, "helm", "mock"),
      reuse: alternatingValue(index, "Yes", "No"),
      status: status(index),
      version,
    };
  });

export const GET: RequestHandler = async ({ url }) => {
  await wait(MIN_RESPONSE_DELAY + Math.random() * RANDOM_RESPONSE_DELAY);

  const countParameter = url.searchParams.get("count");
  let count = DEFAULT_COUNT;
  if (countParameter !== null) {
    const requested = Number(countParameter);
    if (Number.isFinite(requested)) {
      count = Math.min(Math.max(requested, 0), MAX_COUNT);
    }
  }

  return json({
    apiVersion: "star.konfidence.cloud/v1alpha1",
    items: generateArtifactDeployments(count),
    kind: "ArtifactSummaryList",
  });
};
