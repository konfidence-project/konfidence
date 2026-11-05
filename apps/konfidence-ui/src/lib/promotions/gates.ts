import type { Promotion } from "./promotions.ts";

type PromotionGate = NonNullable<Promotion["gates"]>[number];
const APPROVAL_TYPE = "explicit-approval.konfidence.cloud";
const GATE_LABELS: Record<string, string> = {
  "activated-on-source-stage.konfidence.cloud": "Source stage activation",
  "explicit-approval.konfidence.cloud": "Promotion approval",
};

const gateLabel = (gate: PromotionGate): string => GATE_LABELS[gate.type] ?? gate.type;
const gateStateLabel = (gate: PromotionGate): string => gate.state || "Unknown";
const isPendingApproval = (gate: PromotionGate): boolean =>
  gate.type === APPROVAL_TYPE && gate.state === "Pending";
const satisfiedCount = (gates: readonly PromotionGate[]): number =>
  gates.filter((gate) => gate.state === "Satisfied").length;
const gateTone = (gate: PromotionGate): string =>
  ({ Error: "error", Pending: "queued", Satisfied: "healthy" })[gate.state] ?? "queued";

export { gateLabel, gateStateLabel, gateTone, isPendingApproval, satisfiedCount };
export type { PromotionGate };
