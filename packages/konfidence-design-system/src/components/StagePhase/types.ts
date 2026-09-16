type StagePhaseState = "done" | "active" | "failed" | "pending";

interface StagePhaseItem {
  /** Short label shown under the bar (hidden in `compact` size). */
  label: string;
  /** Visual state for this segment. */
  state: StagePhaseState;
}

type StagePhaseSize = "compact" | "default" | "lg";

export type { StagePhaseItem, StagePhaseSize, StagePhaseState };
