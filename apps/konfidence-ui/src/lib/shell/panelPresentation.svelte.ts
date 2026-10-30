interface PanelViewport {
  readonly current: boolean;
}

const DESKTOP_PANEL_MIN_WIDTH = 768;

const createDesktopPanelViewport = (): PanelViewport => {
  let width = $state(typeof globalThis.innerWidth === "number" ? globalThis.innerWidth : 0);
  $effect(() => {
    // oxlint-disable-next-line unicorn/consistent-function-scoping -- updates this instance's reactive width.
    const update = (): void => {
      width = globalThis.innerWidth;
    };
    globalThis.addEventListener("resize", update);
    update();
    return () => globalThis.removeEventListener("resize", update);
  });
  return {
    get current(): boolean {
      return width >= DESKTOP_PANEL_MIN_WIDTH;
    },
  };
};

export { createDesktopPanelViewport };
export type { PanelViewport };
