import type { Vector, VectorList } from "$lib/vectors";
import type { RequestHandler } from "./$types";
import { json } from "@sveltejs/kit";
import { requireUser } from "$lib/server/auth";

const MILLISECONDS_PER_MINUTE = 60_000;
const MIN_RESPONSE_DELAY = 100;
const RANDOM_RESPONSE_DELAY = 500;
const HEX_COLOR_MODULUS = 16_777_215;
const HEX_RADIX = 16;
const HASH_LENGTH = 6;
const VERSION_BUCKET_SIZE = 10;
const MAX_REGION_COUNT = 5;
const MAX_ARTIFACT_COUNT = 12;
const MAX_GENERATION = 20;
const GENERATED_VECTOR_INTERVAL_MINUTES = 7;

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
  Math.abs(Math.sin(seed) * HEX_COLOR_MODULUS)
    .toString(HEX_RADIX)
    .slice(0, HASH_LENGTH)
    .padStart(HASH_LENGTH, "0");

const pick = <Item>(items: Item[], seed: number) => items[seed % items.length];

// Generates `count` synthetic vectors, useful for stress-testing the table
// With large data sets (virtualization + growing).
const generateVectors = (count: number): Vector[] =>
  Array.from({ length: count }, (_value, index) => {
    const major = 2;
    const minor = Math.floor(index / VERSION_BUCKET_SIZE);
    const patch = index % VERSION_BUCKET_SIZE;
    const regionStart = index % REGIONS.length;
    const regionCount = 1 + (index % MAX_REGION_COUNT);

    return vector({
      artifactCount: 1 + (index % MAX_ARTIFACT_COUNT),
      deployedOn: Array.from(
        { length: regionCount },
        (_ignored, offset) => REGIONS[(regionStart + offset) % REGIONS.length],
      ),
      generation: 1 + (index % MAX_GENERATION),
      hash: randomHash(index + 1),
      health: pick(HEALTHS, index),
      minutesAgo: index * GENERATED_VECTOR_INTERVAL_MINUTES,
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

export const GET: RequestHandler = async ({ locals, url }) => {
  requireUser(locals);
  await wait(MIN_RESPONSE_DELAY + Math.random() * RANDOM_RESPONSE_DELAY);

  const requested = Number(url.searchParams.get("count"));
  let count = DEFAULT_COUNT;
  if (Number.isFinite(requested)) {
    count = Math.min(Math.max(requested, 0), MAX_COUNT);
  }

  return json({
    apiVersion: "star.konfidence.cloud/v1alpha1",
    items: buildVectors(count),
    kind: "VectorList",
  } satisfies VectorList);
};
