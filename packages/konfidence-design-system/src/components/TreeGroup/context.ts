import { createContext } from "svelte";

export const [getTreeGroup, setTreeGroup] = createContext<{
  readonly selectedValue: string | undefined;
  select: (value: string) => void;
}>();
