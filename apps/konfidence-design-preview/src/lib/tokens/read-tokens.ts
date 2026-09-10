/**
 * Client-side helpers that read Konfidence design tokens back from
 * their live CSS custom-property values via `getComputedStyle`. The
 * Foundations sections of the gallery use these so the swatches never
 * drift from the tokens.css shipped by @konfidence/design-system.
 */

const readVar = (name: string): string => {
  if (typeof globalThis.document === "undefined") {
    return "";
  }
  const value = globalThis.getComputedStyle(document.documentElement).getPropertyValue(name);
  return value.trim();
};

/**
 * Build a list of `--<prefix>-<step>` variable names for standard ramps
 * (e.g. `amber-50, amber-100, …, amber-900`).
 */
const rampSteps = (
  prefix: string,
  steps: readonly (number | string)[],
): readonly { name: string; step: string; cssVar: string }[] =>
  steps.map((step) => ({
    cssVar: `var(--${prefix}-${step})`,
    name: `--${prefix}-${step}`,
    step: String(step),
  }));

const DEFAULT_RAMP: readonly number[] = [50, 100, 200, 300, 400, 500, 600, 700, 800, 900];

const AMBER_RAMP = rampSteps("amber", DEFAULT_RAMP);
const TEAL_RAMP = rampSteps("teal", DEFAULT_RAMP);
const NEUTRAL_RAMP = rampSteps("neutral", [0, 25, 50, 100, 200, 300, 400, 500, 600, 700, 800, 900]);
const DARK_RAMP = rampSteps("dark", [100, 200, 300, 400, 500, 600, 700, 750, 800, 850, 900]);

const CHART_ACCENTS: readonly { name: string; cssVar: string }[] = [
  { cssVar: "var(--chart-blue)", name: "--chart-blue" },
  { cssVar: "var(--chart-green)", name: "--chart-green" },
  { cssVar: "var(--chart-red)", name: "--chart-red" },
  { cssVar: "var(--chart-orange)", name: "--chart-orange" },
  { cssVar: "var(--chart-purple)", name: "--chart-purple" },
  { cssVar: "var(--chart-cyan)", name: "--chart-cyan" },
  { cssVar: "var(--chart-pink)", name: "--chart-pink" },
  { cssVar: "var(--chart-grey)", name: "--chart-grey" },
];

const STATUS_ROLES: readonly { key: string; fg: string; bg: string; solid: string }[] = [
  {
    bg: "var(--status-healthy-bg)",
    fg: "var(--status-healthy-fg)",
    key: "healthy",
    solid: "var(--status-healthy-solid)",
  },
  {
    bg: "var(--status-warning-bg)",
    fg: "var(--status-warning-fg)",
    key: "warning",
    solid: "var(--status-warning-solid)",
  },
  {
    bg: "var(--status-degraded-bg)",
    fg: "var(--status-degraded-fg)",
    key: "degraded",
    solid: "var(--status-degraded-solid)",
  },
  {
    bg: "var(--status-error-bg)",
    fg: "var(--status-error-fg)",
    key: "error",
    solid: "var(--status-error-solid)",
  },
  {
    bg: "var(--status-promoting-bg)",
    fg: "var(--status-promoting-fg)",
    key: "promoting",
    solid: "var(--status-promoting-solid)",
  },
  {
    bg: "var(--status-deploying-bg)",
    fg: "var(--status-deploying-fg)",
    key: "deploying",
    solid: "var(--status-deploying-solid)",
  },
  {
    bg: "var(--status-queued-bg)",
    fg: "var(--status-queued-fg)",
    key: "queued",
    solid: "var(--status-queued-solid)",
  },
];

const TAG_COLORS: readonly string[] = [
  "red",
  "orange",
  "amber",
  "lime",
  "green",
  "teal",
  "cyan",
  "blue",
  "violet",
  "purple",
  "pink",
  "brown",
];

const GRADIENTS: readonly { name: string; cssVar: string; label: string }[] = [
  { cssVar: "var(--gradient-planet)", label: "Planet", name: "--gradient-planet" },
  { cssVar: "var(--gradient-amber)", label: "Amber", name: "--gradient-amber" },
  { cssVar: "var(--gradient-teal)", label: "Teal", name: "--gradient-teal" },
  { cssVar: "var(--gradient-aurora)", label: "Aurora", name: "--gradient-aurora" },
  { cssVar: "var(--gradient-hero-bg)", label: "Hero background", name: "--gradient-hero-bg" },
  { cssVar: "var(--gradient-section)", label: "Section", name: "--gradient-section" },
  { cssVar: "var(--gradient-tile)", label: "Tile", name: "--gradient-tile" },
];

const TYPE_SCALE: readonly { name: string; cssVar: string; sample: string }[] = [
  { cssVar: "var(--text-hero)", name: "--text-hero", sample: "Hero" },
  { cssVar: "var(--text-display)", name: "--text-display", sample: "Display" },
  { cssVar: "var(--text-h1)", name: "--text-h1", sample: "Heading 1" },
  { cssVar: "var(--text-h2)", name: "--text-h2", sample: "Heading 2" },
  { cssVar: "var(--text-h3)", name: "--text-h3", sample: "Heading 3" },
  { cssVar: "var(--text-body)", name: "--text-body", sample: "Body text" },
  { cssVar: "var(--text-sm)", name: "--text-sm", sample: "Small text" },
  { cssVar: "var(--text-meta)", name: "--text-meta", sample: "META" },
];

const SPACING_STEPS: readonly (number | string)[] = [1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 20, 24];

const RADIUS_STEPS: readonly (number | string)[] = [4, 6, 10, 14, 20, 28, "pill"];

export {
  AMBER_RAMP,
  CHART_ACCENTS,
  DARK_RAMP,
  DEFAULT_RAMP,
  GRADIENTS,
  NEUTRAL_RAMP,
  RADIUS_STEPS,
  readVar,
  SPACING_STEPS,
  STATUS_ROLES,
  TAG_COLORS,
  TEAL_RAMP,
  TYPE_SCALE,
};
