import { createContext } from "svelte";

import type { StageCardVariant } from "$lib/components/StageCard.svelte";

const STORAGE_KEY = "konfidence.ui.stageCardVariant";

const variants = new Set<StageCardVariant>(["fiori", "fiori-mockup", "custom"]);

function isStageCardVariant(value: string | null): value is StageCardVariant {
  return variants.has(value as StageCardVariant);
}

function load(): StageCardVariant {
  const storedVariant = localStorage.getItem(STORAGE_KEY);
  return isStageCardVariant(storedVariant) ? storedVariant : "custom";
}

export class StageCardVariantPreference {
  selected = $state<StageCardVariant>(load());

  constructor() {
    $effect(() => {
      localStorage.setItem(STORAGE_KEY, this.selected);
    });
  }

  select(value: string) {
    if (isStageCardVariant(value)) {
      this.selected = value;
    }
  }
}

export const [getStageCardVariantPreference, setStageCardVariantPreference] =
  createContext<StageCardVariantPreference>();
