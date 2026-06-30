// Note: global store like e.g. uiPreferences.svelte.ts could also be an option

const STORAGE_KEY = "konfidence.ui.sidebarMode";
export type SidebarMode = "Expanded" | "Collapsed";

function load(): SidebarMode {
    return localStorage.getItem(STORAGE_KEY) === "Collapsed"
        ? "Collapsed"
        : "Expanded";
}

export const sidebar = $state<{ mode: SidebarMode }>({ mode: load() });

$effect.root(() => {
    $effect(() => {
        localStorage.setItem(STORAGE_KEY, sidebar.mode);
    });
});

export function toggleSidebar() {
    sidebar.mode = sidebar.mode === "Expanded" ? "Collapsed" : "Expanded";
}