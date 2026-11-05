import { getContext, setContext } from "svelte";

interface GroupSelection {
  readonly selectedValue: string | undefined;
  select: (value: string) => boolean;
}

const selectionKey = Symbol("ExpandableGroup selection");

const setGroupSelection = (selection: GroupSelection): void => {
  setContext(selectionKey, selection);
};

const getGroupSelection = (): GroupSelection | undefined =>
  getContext<GroupSelection | undefined>(selectionKey);

export { getGroupSelection, setGroupSelection };
