export const clampPanelSize = (size: number, min: number, max: number): number =>
  Math.min(max, Math.max(min, Math.round(size)));
