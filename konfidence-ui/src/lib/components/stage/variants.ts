type StageCardVariant = "fiori" | "fiori-mockup" | "custom";

interface StageCardVariantDefinition {
  descriptionKey: string;
  id: StageCardVariant;
  labelKey: string;
}

const STAGE_CARD_VARIANTS: StageCardVariantDefinition[] = [
  {
    descriptionKey: "STAGE_VARIANT_FIORI_DESC",
    id: "fiori",
    labelKey: "STAGE_VARIANT_FIORI_LABEL",
  },
  {
    descriptionKey: "STAGE_VARIANT_FIORI_MOCKUP_DESC",
    id: "fiori-mockup",
    labelKey: "STAGE_VARIANT_FIORI_MOCKUP_LABEL",
  },
  {
    descriptionKey: "STAGE_VARIANT_CUSTOM_DESC",
    id: "custom",
    labelKey: "STAGE_VARIANT_CUSTOM_LABEL",
  },
];

export { STAGE_CARD_VARIANTS };
export type { StageCardVariant, StageCardVariantDefinition };
