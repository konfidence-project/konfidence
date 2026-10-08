import type { PromotionEdgeStatus } from "$lib/landscape/landscapeGraph";

// Badge tones for the promotion lifecycle. Each maps to a `--status-<tone>-*`
// family in the design-system token sheet.
type PromotionBadgeTone =
  | "deploying"
  | "degraded"
  | "error"
  | "healthy"
  | "promoting"
  | "queued"
  | "warning";

interface PromotionStatusEntry {
  badge: PromotionBadgeTone;
  label: string;
  // Marching-dashes animation for genuinely in-flight promotions.
  animated: boolean;
  // Full, literal Tailwind class strings per tone. They are spelled out in full
  // (never interpolated) so Tailwind's static scanner keeps them in the build.
  stroke: string;
  arrowFill: string;
  pill: string;
  dot: string;
}

// Literal class strings keyed by tone. Interpolating `--status-${tone}-*` would
// hide the utilities from Tailwind's scanner, so every variant is written out.
const toneClasses: Record<
  PromotionBadgeTone,
  Pick<PromotionStatusEntry, "arrowFill" | "dot" | "pill" | "stroke">
> = {
  degraded: {
    arrowFill: "fill-[var(--status-degraded-solid)]",
    dot: "bg-[var(--status-degraded-solid)]",
    pill: "text-[var(--status-degraded-fg)] bg-[var(--status-degraded-bg)]",
    stroke: "stroke-[var(--status-degraded-solid)]!",
  },
  deploying: {
    arrowFill: "fill-[var(--status-deploying-solid)]",
    dot: "bg-[var(--status-deploying-solid)]",
    pill: "text-[var(--status-deploying-fg)] bg-[var(--status-deploying-bg)]",
    stroke: "stroke-[var(--status-deploying-solid)]!",
  },
  error: {
    arrowFill: "fill-[var(--status-error-solid)]",
    dot: "bg-[var(--status-error-solid)]",
    pill: "text-[var(--status-error-fg)] bg-[var(--status-error-bg)]",
    stroke: "stroke-[var(--status-error-solid)]!",
  },
  healthy: {
    arrowFill: "fill-[var(--status-healthy-solid)]",
    dot: "bg-[var(--status-healthy-solid)]",
    pill: "text-[var(--status-healthy-fg)] bg-[var(--status-healthy-bg)]",
    stroke: "stroke-[var(--status-healthy-solid)]!",
  },
  promoting: {
    arrowFill: "fill-[var(--status-promoting-solid)]",
    dot: "bg-[var(--status-promoting-solid)]",
    pill: "text-[var(--status-promoting-fg)] bg-[var(--status-promoting-bg)]",
    stroke: "stroke-[var(--status-promoting-solid)]!",
  },
  queued: {
    arrowFill: "fill-[var(--status-queued-solid)]",
    dot: "bg-[var(--status-queued-solid)]",
    pill: "text-[var(--status-queued-fg)] bg-[var(--status-queued-bg)]",
    stroke: "stroke-[var(--status-queued-solid)]!",
  },
  warning: {
    arrowFill: "fill-[var(--status-warning-solid)]",
    dot: "bg-[var(--status-warning-solid)]",
    pill: "text-[var(--status-warning-fg)] bg-[var(--status-warning-bg)]",
    stroke: "stroke-[var(--status-warning-solid)]!",
  },
};

const entry = (
  badge: PromotionBadgeTone,
  label: string,
  animated = false,
): PromotionStatusEntry => ({ animated, badge, label, ...toneClasses[badge] });

// Mirrors api/v1alpha1/vectorpromotion_types.go (VectorPromotionState). Every
// lifecycle state has a distinct colour and label so an edge is never
// ambiguous. Only InProgress animates (genuinely moving work).
const promotionStatuses: Record<PromotionEdgeStatus, PromotionStatusEntry> = {
  Blocked: entry("degraded", "Blocked"),
  Failed: entry("error", "Failed"),
  InProgress: entry("deploying", "In progress", true),
  Ready: entry("promoting", "Ready"),
  Succeeded: entry("healthy", "Succeeded"),
  Superseded: entry("queued", "Superseded"),
  Waiting: entry("warning", "Waiting"),
};

export { promotionStatuses };
export type { PromotionBadgeTone, PromotionStatusEntry };
