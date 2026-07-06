import type { StageCardVariant } from "$lib/components/stage/variants.js";
import { createContext } from "svelte";

const STORAGE_KEY = "konfidence.ui.stageCardVariant";
const variants = new Set<StageCardVariant>(["fiori", "fiori-mockup", "custom"]);

const isStageCardVariant = (value: string | undefined): value is StageCardVariant =>
  variants.has(value as StageCardVariant);

const load = (): StageCardVariant => {
  const storedVariant = globalThis.localStorage?.getItem(STORAGE_KEY) ?? undefined;
  if (isStageCardVariant(storedVariant)) {
    return storedVariant;
  }

  return "custom";
};

class StageCardVariantPreference {
  selected = $state<StageCardVariant>(load());

  constructor() {
    $effect(() => {
      globalThis.localStorage?.setItem(STORAGE_KEY, this.selected);
    });
  }

  select(value: string) {
    if (isStageCardVariant(value)) {
      this.selected = value;
    }
  }
}

const [getStageCardVariantPreference, setStageCardVariantPreference] =
  createContext<StageCardVariantPreference>();

export { getStageCardVariantPreference, setStageCardVariantPreference, StageCardVariantPreference };
