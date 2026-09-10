import type { Mode } from "./types.js";

const isDarkMode = (mode: Mode, prefersDark = getPrefersDark()): boolean =>
  mode === "dark" || (mode === "system" && prefersDark);

const getPrefersDark = (): boolean =>
  typeof globalThis.matchMedia === "function" &&
  globalThis.matchMedia("(prefers-color-scheme: dark)").matches;

export { isDarkMode };
