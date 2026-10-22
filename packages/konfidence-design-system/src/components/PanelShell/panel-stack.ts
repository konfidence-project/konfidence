const nonModalPanels: symbol[] = [];

const registerNonModalPanel = (panel: symbol): void => {
  nonModalPanels.push(panel);
};

const unregisterNonModalPanel = (panel: symbol): void => {
  const index = nonModalPanels.lastIndexOf(panel);
  if (index !== -1) {
    nonModalPanels.splice(index, 1);
  }
};

const isTopmostNonModalPanel = (panel: symbol): boolean => nonModalPanels.at(-1) === panel;

export { isTopmostNonModalPanel, registerNonModalPanel, unregisterNonModalPanel };
