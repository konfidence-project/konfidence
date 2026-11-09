/**
 * Shared, persisted width for every design-system SidePanel in the app.
 *
 * The design system does not own storage. `undefined` means no stored
 * preference, so the panel falls back to its own default.
 *
 * The width storage key is preserved from the previous `sidePanelWidth` store
 * so returning users keep their saved right-panel width.
 */
const STORAGE_KEY = "konfidence:side-panel-width";

const readStored = (): number | undefined => {
  try {
    const raw = globalThis.localStorage?.getItem(STORAGE_KEY);
    if (raw === null || raw === undefined) {
      return undefined;
    }
    const value = Number.parseInt(raw, 10);
    return Number.isFinite(value) && value > 0 ? value : undefined;
  } catch {
    return undefined;
  }
};

const writeStored = (value: number): void => {
  try {
    globalThis.localStorage?.setItem(STORAGE_KEY, String(Math.round(value)));
  } catch {
    return;
  }
};

class PanelDimensions {
  #width = $state<number | undefined>(readStored());

  get width(): number | undefined {
    return this.#width;
  }

  setWidth = (width: number): void => {
    this.#width = width;
    writeStored(width);
  };
}

const panelDimensions = new PanelDimensions();

export { PanelDimensions, panelDimensions, STORAGE_KEY };
