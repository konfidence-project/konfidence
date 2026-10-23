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
}

const outsidePanelDismiss = (
  node: HTMLElement,
  options: OutsidePanelDismissOptions,
): { destroy: () => void } => {
  const handlePointerDown = (event: PointerEvent): void => {
    if (options.enabled() && isOutsidePanelPointerActivation(event)) {
      options.dismiss();
    }
  };
  const handleClick = (event: MouseEvent): void => {
    if (options.enabled() && isOutsidePanelKeyboardActivation(event)) {
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
