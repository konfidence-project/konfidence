/**
 * Shared, persisted width for the design-system `SidePanel` overlay.
 *
 * The design system keeps the `SidePanel` stateless about storage, so this app
 * owns the single shared width used by every `SidePanel` instance (stage
 * overlay, vector deployments, artifact deployments), persisted to
 * `localStorage`. `undefined` means no stored preference, so the `SidePanel`
 * falls back to its own default width.
 */
const STORAGE_KEY = "konfidence:side-panel-width";

const readStoredWidth = (): number | undefined => {
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

const writeStoredWidth = (width: number): void => {
  try {
    globalThis.localStorage?.setItem(STORAGE_KEY, String(Math.round(width)));
  } catch {
    return;
  }
};

class SidePanelWidth {
  #width = $state<number | undefined>(readStoredWidth());

  get current(): number | undefined {
    return this.#width;
  }

  set = (width: number): void => {
    this.#width = width;
    writeStoredWidth(width);
  };
}

const sidePanelWidth = new SidePanelWidth();

export { STORAGE_KEY, SidePanelWidth, sidePanelWidth };
