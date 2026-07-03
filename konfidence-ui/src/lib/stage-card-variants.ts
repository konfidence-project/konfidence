export type StageCardVariant = "fiori" | "fiori-mockup" | "custom";

export const STAGE_CARD_VARIANTS: {
  id: StageCardVariant;
  label: string;
  description: string;
}[] = [
  {
    id: "fiori",
    label: "Fiori",
    description: "Pure UI5 Web Components; inherits SAP theming.",
  },
  {
    id: "fiori-mockup",
    label: "Fiori · Mockup",
    description: "UI5 primitives (Card, Menu, Tag, Icon) laid out to match the Konfidence mockup.",
  },
  {
    id: "custom",
    label: "Custom",
    description: "No UI5 wc — hand-rolled markup, closest to the mockup.",
  },
];
