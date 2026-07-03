type StageCardVariant = "fiori" | "fiori-mockup" | "custom";

const STAGE_CARD_VARIANTS: {
  description: string;
  id: StageCardVariant;
  label: string;
}[] = [
  {
    description: "Pure UI5 Web Components; inherits SAP theming.",
    id: "fiori",
    label: "Fiori",
  },
  {
    description: "UI5 primitives (Card, Menu, Tag, Icon) laid out to match the Konfidence mockup.",
    id: "fiori-mockup",
    label: "Fiori · Mockup",
  },
  {
    description: "No UI5 wc — hand-rolled markup, closest to the mockup.",
    id: "custom",
    label: "Custom",
  },
];

export { STAGE_CARD_VARIANTS };
export type { StageCardVariant };
