// Note: global store like e.g. uiPreferences.svelte.ts could also be an option.

const STORAGE_KEY = "konfidence.ui.sidebarMode";

type SidebarMode = "Expanded" | "Collapsed";

const load = (): SidebarMode => {
  if (globalThis.localStorage?.getItem(STORAGE_KEY) === "Collapsed") {
    return "Collapsed";
  }

  return "Expanded";
};

const sidebar = $state<{ mode: SidebarMode }>({ mode: load() });

$effect.root(() => {
  $effect(() => {
    globalThis.localStorage?.setItem(STORAGE_KEY, sidebar.mode);
  });
});

const toggleSidebar = () => {
  if (sidebar.mode === "Expanded") {
    sidebar.mode = "Collapsed";
    return;
  }

  sidebar.mode = "Expanded";
};

export { sidebar, toggleSidebar };
export type { SidebarMode };
