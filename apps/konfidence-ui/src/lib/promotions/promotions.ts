import type { Landscape, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";

type Promotion = VectorPromotionConfig["promotions"][number];
type PromotionReference = Promotion["source"] | Promotion["target"];

const EXPLICIT_APPROVAL_GATE = "explicit-approval.konfidence.cloud";
const approvalGate = (promotion: Promotion) =>
  promotion.gates?.find((gate) => gate.type === EXPLICIT_APPROVAL_GATE);

// The current gate API replaces the plan's legacy requireApproval/approval fields.
// Be conservative: only a recorded Pending explicit-approval gate is actionable.
const isActionable = (promotion: Promotion): boolean =>
  promotion.status === "Waiting" && approvalGate(promotion)?.state === "Pending";

const statusLabel = (promotion: Promotion): string =>
  promotion.status === "InProgress" ? "In progress" : (promotion.status ?? "Unknown");

const statusTone = (promotion: Promotion): string => {
  switch (String(promotion.status)) {
    case "Waiting": {
      return "queued";
    }
    case "Ready": {
      return "queued";
    }
    case "InProgress": {
      return "promoting";
    }
    case "Blocked":
    case "Failed": {
      return "error";
    }
    case "Succeeded": {
      return "healthy";
    }
    default: {
      return "unknown";
    }
  }
};

const referenceLabel = (
  reference: PromotionReference,
  landscapes: readonly Landscape[],
): string => {
  if (reference.kind === "VectorTemplate") {
    return `VectorTemplate: ${reference.name}`;
  }
  const landscape = landscapes.find((item) => item.id === reference.landscape);
  return `${landscape?.name ?? reference.landscape ?? "Unresolved landscape"}/${reference.name}`;
};

interface PromotionGroup {
  config: VectorPromotionConfig;
  promotions: readonly Promotion[];
  primary: Promotion | undefined;
  waiting: number;
}

const toPromotionGroup = (
  config: VectorPromotionConfig,
  promotions: readonly Promotion[] = config.promotions,
): PromotionGroup => {
  const ordered = promotions.toSorted(
    (left, right) =>
      (right.sequence ?? -Infinity) - (left.sequence ?? -Infinity) ||
      left.id.localeCompare(right.id),
  );
  const primary = ordered.find((promotion) => promotion.status === "Waiting") ?? ordered[0];
  return {
    config,
    primary,
    promotions: ordered,
    waiting: ordered.filter((promotion) => promotion.status === "Waiting").length,
  };
};

const landscapeLabels = (
  config: VectorPromotionConfig,
  landscapes: readonly Landscape[],
): { id: string; label: string }[] =>
  [
    ...new Set(
      [config.source, config.target]
        .filter((reference) => reference.kind === "Stage")
        .flatMap((reference) => (reference.landscape ? [reference.landscape] : [])),
    ),
  ].map((id) => ({ id, label: landscapes.find((landscape) => landscape.id === id)?.name ?? id }));

const toPromotionGroups = (configs: readonly VectorPromotionConfig[]): PromotionGroup[] =>
  configs
    .toSorted((left, right) => left.id.localeCompare(right.id))
    .map((config) => toPromotionGroup(config));

export {
  approvalGate,
  isActionable,
  landscapeLabels,
  referenceLabel,
  statusLabel,
  statusTone,
  toPromotionGroup,
  toPromotionGroups,
};
export type { Promotion, PromotionGroup, PromotionReference };
