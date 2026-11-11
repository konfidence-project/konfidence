import type { Promotion } from "./promotions.ts";

type PromotionGate = NonNullable<Promotion["gates"]>[number];
const EXPLICIT_APPROVAL_GATE = "explicit-approval.konfidence.cloud";
const GATE_LABELS: Record<string, string> = {
  [EXPLICIT_APPROVAL_GATE]: "Promotion approval",
  "activated-on-source-stage.konfidence.cloud": "Source stage activation",
};

const gateLabel = (gate: PromotionGate): string => GATE_LABELS[gate.type] ?? gate.type;
const gateStateLabel = (gate: PromotionGate): string => gate.state || "Unknown";
const isPendingApproval = (gate: PromotionGate): boolean =>
  gate.type === EXPLICIT_APPROVAL_GATE && gate.state === "Pending";
const satisfiedCount = (gates: readonly PromotionGate[]): number =>
  gates.filter((gate) => gate.state === "Satisfied").length;
const gateTone = (gate: PromotionGate): string =>
  ({ Error: "error", Pending: "queued", Satisfied: "healthy" })[gate.state] ?? "queued";

export {
  EXPLICIT_APPROVAL_GATE,
  gateLabel,
  gateStateLabel,
  gateTone,
  isPendingApproval,
  satisfiedCount,
};
export type { PromotionGate };
