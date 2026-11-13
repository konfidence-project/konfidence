import type { Landscape, VectorPromotionConfig } from "#lib/konfidence-api/types.ts";
import { EXPLICIT_APPROVAL_GATE } from "./gates.ts";

type Promotion = VectorPromotionConfig["promotions"][number];
type PromotionReference = Promotion["source"] | Promotion["target"];

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

const landscapeLabel = (
  id: Landscape["id"] | undefined,
  landscapes: readonly Landscape[],
): string => {
  const landscape = landscapes.find((item) => item.id === id);
  if (!landscape) {
    return id ?? "Unresolved landscape";
  }
  const nameIsShared = landscapes.some(
    (item) => item.id !== landscape.id && item.name === landscape.name,
  );
  return nameIsShared ? `${landscape.name} (${landscape.id})` : landscape.name;
};

const referenceLabel = (
  reference: PromotionReference,
  landscapes: readonly Landscape[],
): string => {
  if (reference.kind === "VectorTemplate") {
    return `VectorTemplate: ${reference.name}`;
  }
  return `${landscapeLabel(reference.landscape, landscapes)}/${reference.name}`;
};

interface PromotionGroup {
  config: VectorPromotionConfig;
  promotions: readonly Promotion[];
  latestWaiting: Promotion | undefined;
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
  const latestWaiting = ordered.find((promotion) => promotion.status === "Waiting");
  return {
    config,
    latestWaiting,
    promotions: ordered,
    waiting: ordered.filter((promotion) => promotion.status === "Waiting").length,
  };
};

export {
  approvalGate,
  isActionable,
  landscapeLabel,
  referenceLabel,
  statusLabel,
  statusTone,
  toPromotionGroup,
};
export type { Promotion, PromotionGroup, PromotionReference };
