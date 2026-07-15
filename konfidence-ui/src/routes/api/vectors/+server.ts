import type { Vector, VectorList } from "$lib/vectors";
import { json } from "@sveltejs/kit";
import type { RequestHandler } from "./$types";

const MILLISECONDS_PER_MINUTE = 60_000;
const MIN_RESPONSE_DELAY = 100;
const RANDOM_RESPONSE_DELAY = 500;

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const timestamp = (minutesAgo: number) =>
  new Date(Date.now() - minutesAgo * MILLISECONDS_PER_MINUTE).toISOString();

interface VectorFixture {
  artifactCount: number;
  deployedOn: string[];
  generation: number;
  hash: string;
  health: Vector["status"]["health"];
  minutesAgo: number;
  name: string;
  namespace: string;
}

const vector = ({
  artifactCount,
  deployedOn,
  generation,
  hash,
  health,
  minutesAgo,
  name,
  namespace,
}: VectorFixture): Vector => ({
  apiVersion: "star.konfidence.cloud/v1alpha1",
  kind: "Vector",
  metadata: {
    creationTimestamp: timestamp(minutesAgo),
    generation,
    name,
    namespace,
  },
  spec: {
    artifactCount,
    deployedOn,
    hash,
    vector: `https://registry.kdenv.lab/konfidence.project/vector:${name}-${hash}`,
  },
  status: {
    health,
  },
});

const baseVectors: Vector[] = [
  vector({
    artifactCount: 7,
    deployedOn: ["dev-us30", "dev-eu12", "dev-jp20", "test-us30", "test-load"],
    generation: 14,
    hash: "a3f2c9",
    health: "Healthy",
    minutesAgo: 12,
    name: "v2.14.0",
    namespace: "default",
  }),
  vector({
    artifactCount: 2,
    deployedOn: ["dev-cn10", "test-cn10"],
    generation: 13,
    hash: "b1c4d7",
    health: "Healthy",
    minutesAgo: 45,
    name: "v2.13.0",
    namespace: "default",
  }),
  vector({
    artifactCount: 2,
    deployedOn: ["perf-us30", "prod-ap30", "prod-jp20"],
    generation: 12,
    hash: "c44e21",
    health: "Error",
    minutesAgo: 180,
    name: "v2.12.0",
    namespace: "default",
  }),
  vector({
    artifactCount: 2,
    deployedOn: ["test-jp20", "prod-eu30", "prod-us30", "prod-br10"],
    generation: 13,
    hash: "f8e1b2",
    health: "Warning",
    minutesAgo: 240,
    name: "v2.13.1",
    namespace: "default",
  }),
  vector({
    artifactCount: 2,
    deployedOn: ["tsm-prod"],
    generation: 11,
    hash: "1b4a22",
    health: "Error",
    minutesAgo: 720,
    name: "v2.11.3",
    namespace: "default",
  }),
  vector({
    artifactCount: 3,
    deployedOn: ["prod-cn10"],
    generation: 12,
    hash: "d9a1f4",
    health: "Healthy",
    minutesAgo: 300,
    name: "v2.12.3",
    namespace: "default",
  }),
];

const HEALTHS: Vector["status"]["health"][] = ["Healthy", "Warning", "Error"];
const REGIONS = [
  "dev-us30",
  "dev-eu12",
  "dev-jp20",
  "test-us30",
  "test-cn10",
  "perf-us30",
  "prod-ap30",
  "prod-eu30",
  "prod-us30",
  "prod-br10",
  "prod-cn10",
];

const randomHash = (seed: number) =>
  Math.abs(Math.sin(seed) * 0xffffff)
    .toString(16)
    .slice(0, 6)
    .padStart(6, "0");

const pick = <T>(items: T[], seed: number) => items[seed % items.length];

// Generates `count` synthetic vectors, useful for stress-testing the table
// with large data sets (virtualization + growing).
const generateVectors = (count: number): Vector[] =>
  Array.from({ length: count }, (_, index) => {
    const major = 2;
    const minor = Math.floor(index / 10);
    const patch = index % 10;
    const regionStart = index % REGIONS.length;
    const regionCount = 1 + (index % 5);

    return vector({
      artifactCount: 1 + (index % 12),
      deployedOn: Array.from(
        { length: regionCount },
        (_, offset) => REGIONS[(regionStart + offset) % REGIONS.length],
      ),
      generation: 1 + (index % 20),
      hash: randomHash(index + 1),
      health: pick(HEALTHS, index),
      minutesAgo: index * 7,
      name: `v${major}.${minor}.${patch}-gen${index}`,
      namespace: "default",
    });
  });

const buildVectors = (count: number): Vector[] => {
  if (count <= baseVectors.length) {
    return baseVectors.slice(0, count);
  }
  return [...baseVectors, ...generateVectors(count - baseVectors.length)];
};

const DEFAULT_COUNT = baseVectors.length;
const MAX_COUNT = 100_000;

export const GET: RequestHandler = async ({ url }) => {
  await wait(MIN_RESPONSE_DELAY + Math.random() * RANDOM_RESPONSE_DELAY);

  const requested = Number(url.searchParams.get("count"));
  const count = Number.isFinite(requested)
    ? Math.min(Math.max(requested, 0), MAX_COUNT)
    : DEFAULT_COUNT;

  return json({
    apiVersion: "star.konfidence.cloud/v1alpha1",
    items: buildVectors(count),
    kind: "VectorList",
  } satisfies VectorList);
};
