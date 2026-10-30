/**
 * Shared, persisted dimensions for the design-system overlay panels.
 *
 * The design system keeps `SidePanel` and `BottomSheet` stateless about
 * storage, so this app owns the single shared dimension used by every panel
 * instance, persisted to `localStorage`. Dimensions are keyed by placement:
 * the right panel persists its `width`, the bottom panel persists its
 * `height`. `undefined` means no stored preference, so the panel falls back to
 * its own default.
 *
 * The width storage key is preserved from the previous `sidePanelWidth` store
 * so returning users keep their saved right-panel width.
 */
type PanelPlacement = "right" | "bottom";

const STORAGE_KEYS: Record<PanelPlacement, string> = {
  bottom: "konfidence:bottom-sheet-height",
  right: "konfidence:side-panel-width",
};

// The right-panel width key, preserved from the former `sidePanelWidth` store so
// returning users keep their saved width. Exposed for the migration regression test.
const STORAGE_KEY = STORAGE_KEYS.right;

const readStored = (placement: PanelPlacement): number | undefined => {
  try {
    const raw = globalThis.localStorage?.getItem(STORAGE_KEYS[placement]);
    if (raw === null || raw === undefined) {
      return undefined;
    }
    const value = Number.parseInt(raw, 10);
    return Number.isFinite(value) && value > 0 ? value : undefined;
  } catch {
    return undefined;
  }
};

const writeStored = (placement: PanelPlacement, value: number): void => {
  try {
    globalThis.localStorage?.setItem(STORAGE_KEYS[placement], String(Math.round(value)));
  } catch {
    return;
  }
};

class PanelDimensions {
  #width = $state<number | undefined>(readStored("right"));
  #height = $state<number | undefined>(readStored("bottom"));

  get width(): number | undefined {
    return this.#width;
  }

  get height(): number | undefined {
    return this.#height;
  }

  setWidth = (width: number): void => {
    this.#width = width;
    writeStored("right", width);
  };

  setHeight = (height: number): void => {
    this.#height = height;
    writeStored("bottom", height);
  };
}

const panelDimensions = new PanelDimensions();

export { PanelDimensions, panelDimensions, STORAGE_KEY, STORAGE_KEYS };
export type { PanelPlacement };
