import type { VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
import type { Promotion } from "#lib/promotions/promotions.ts";

/* eslint-disable no-magic-numbers -- Test scenarios encode explicit promotion sequence order. */

const promotion = (
  id: string,
  sequence: number,
  overrides: Partial<Promotion> = {},
): Promotion => ({
  gates: [{ state: "Pending", type: "explicit-approval.konfidence.cloud" }],
  id,
  sequence,
  source: { kind: "Stage", landscape: "development", name: "dev" },
  status: "Waiting",
  target: { kind: "Stage", landscape: "production", name: "prod" },
  vector: `vector-${id}`,
  ...overrides,
});

const configs: VectorPromotionConfig[] = [
  {
    id: "z-template",
    promotions: [
      promotion("z-newest", 4, { gates: [], status: "Succeeded" }),
      promotion("z-older", 3, { gates: [], status: "Superseded" }),
    ],
    source: { kind: "VectorTemplate", name: "release-template" },
    target: { kind: "Stage", landscape: "production", name: "prod" },
  },
  {
    id: "a-flow",
    promotions: [
      promotion("a-newest", 10, { gates: [], status: "Ready" }),
      promotion("a-waiting", 9),
      promotion("a-older-waiting", 8),
      promotion("a-history", 7, {
        gates: [
          {
            setBy: "approver@example.com",
            state: "Satisfied",
            type: "explicit-approval.konfidence.cloud",
          },
        ],
        status: "Succeeded",
      }),
    ],
    source: { kind: "Stage", landscape: "development", name: "dev" },
    target: { kind: "Stage", landscape: "production", name: "prod" },
  },
  {
    id: "empty-flow",
    promotions: [],
    source: { kind: "VectorTemplate", name: "empty-template" },
    target: { kind: "Stage", landscape: "production", name: "prod" },
  },
];

export { configs, promotion };
