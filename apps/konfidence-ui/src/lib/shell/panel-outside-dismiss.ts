const isPanelOrRowInteraction = (event: Event): boolean =>
  event
    .composedPath()
    .some(
      (target) =>
        target instanceof Element && target.closest("[data-panel-surface], [data-row-id]") !== null,
    );

const isOutsidePanelPointerActivation = (event: PointerEvent): boolean =>
  event.button === 0 && !isPanelOrRowInteraction(event);

const isOutsidePanelKeyboardActivation = (event: MouseEvent): boolean =>
  event.detail === 0 && !isPanelOrRowInteraction(event);

interface OutsidePanelDismissOptions {
  dismiss: () => void;
  enabled: () => boolean;
  /** Feature controls that remain interactive without dismissing their panel. */
  ignoreSelector?: string;
}

const outsidePanelDismiss = (
  node: HTMLElement,
  options: OutsidePanelDismissOptions,
): { destroy: () => void } => {
  const isIgnored = (event: Event): boolean => {
    const selector = options.ignoreSelector;
    return (
      selector !== undefined &&
      event
        .composedPath()
        .some((target) => target instanceof Element && target.closest(selector) !== null)
    );
  };
  const handlePointerDown = (event: PointerEvent): void => {
    if (options.enabled() && !isIgnored(event) && isOutsidePanelPointerActivation(event)) {
      options.dismiss();
    }
  };
  const handleClick = (event: MouseEvent): void => {
    if (options.enabled() && !isIgnored(event) && isOutsidePanelKeyboardActivation(event)) {
      options.dismiss();
    }
  };
  const target = node.ownerDocument;
  target.addEventListener("pointerdown", handlePointerDown, true);
  target.addEventListener("click", handleClick, true);
  return {
    destroy: () => {
      target.removeEventListener("pointerdown", handlePointerDown, true);
      target.removeEventListener("click", handleClick, true);
    },
  };
};

export { outsidePanelDismiss };
export type { OutsidePanelDismissOptions };
