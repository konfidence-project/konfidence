type ChangeState = "added" | "changed" | "removed";
type ChangeValue = string | boolean;

interface ChangeListItem {
  /** Stable identity within the list. */
  id: string;
  name: string;
  state: ChangeState;
  /** Optional visible status label, e.g. Enabled or Disabled. */
  label?: string;
  oldValue?: ChangeValue;
  newValue?: ChangeValue;
  /** Starts a new labelled section when different from the previous item. */
  group?: string;
}

export type { ChangeListItem, ChangeState, ChangeValue };
