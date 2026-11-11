import { getContext, setContext } from "svelte";

interface GroupSelection {
  readonly selectedValue: string | undefined;
  select: (value: string) => void;
}

const selectionKey = Symbol("ExpandableGroup selection");

const setGroupSelection = (selection: GroupSelection): void => {
  setContext(selectionKey, selection);
};

const getGroupSelection = (): GroupSelection => {
  const selection = getContext<GroupSelection | undefined>(selectionKey);
  if (!selection) {
    throw new Error("SelectableRow must be rendered inside an ExpandableGroup.");
  }
  return selection;
};

export { getGroupSelection, setGroupSelection };
