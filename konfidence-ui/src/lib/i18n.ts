import { addMessages, init } from "svelte-i18n";
import { DEFAULT_LOCALE } from "$lib/locale";

/* oxlint-disable sort-keys -- Keep message groups in the order users encounter them. */
const en = {
  shell: {
    accountMenu: "Account menu",
    applicationOverlays: "Application overlays",
    closeNavigation: "Close navigation",
    delivery: "Delivery",
    home: "Konfidence home",
    landscape: "Landscape",
    notifications: "Notifications, 3 unread",
    openAccountMenu: "Open account menu for {name}",
    openNavigation: "Open navigation",
    projectNavigation: "Project navigation",
    settings: "Settings",
    signOut: "Sign Out",
    vectorDeployments: "Vector Deployments",
  },
  settings: {
    account: {
      email: "Email",
      familyName: "Family name",
      givenName: "Given name",
      noRoles: "(none)",
      roles: "Roles",
    },
    appearance: "Appearance",
    appearanceDescription: "Choose how Konfidence looks on this device.",
    avatarFor: "Avatar for {name}",
    close: "Close settings",
    description: "Manage your profile display, application theme, and language.",
    language: "Language",
    profile: "Profile",
    sections: "Settings sections",
    theme: "Theme",
    themes: {
      konfidence: { description: "Bright and focused", label: "Konfidence" },
      "konfidence-dark": { description: "Low-light workspace", label: "Konfidence Dark" },
      sap_horizon: { description: "Familiar enterprise blue", label: "Horizon" },
    },
    title: "Settings",
  },
  landscape: {
    canvas: "Stage landscape",
    stages: "Delivery stages",
    title: "Delivery landscape",
  },
  vector: {
    columns: {
      deployment: "Vector deployment",
      landscape: "Landscape",
      repository: "Repository",
      stage: "Stage",
      status: "Status",
      version: "Version",
    },
    empty: "No vector deployments match your search.",
    inventory: "Project inventory",
    search: "Search vector deployments",
    searchPlaceholder: "Search vector deployments...",
    summary:
      "Versioned vectors currently assigned to project stages. Showing {visible} of {total}.",
    table: "Vector deployments",
    title: "Vector Deployments",
  },
};

const de = {
  shell: {
    accountMenu: "Kontomenü",
    applicationOverlays: "Anwendungsdialoge",
    closeNavigation: "Navigation schließen",
    delivery: "Auslieferung",
    home: "Konfidence Startseite",
    landscape: "Landschaft",
    notifications: "Benachrichtigungen, 3 ungelesen",
    openAccountMenu: "Kontomenü für {name} öffnen",
    openNavigation: "Navigation öffnen",
    projectNavigation: "Projektnavigation",
    settings: "Einstellungen",
    signOut: "Abmelden",
    vectorDeployments: "Vektorbereitstellungen",
  },
  settings: {
    account: {
      email: "E-Mail",
      familyName: "Nachname",
      givenName: "Vorname",
      noRoles: "(keine)",
      roles: "Rollen",
    },
    appearance: "Darstellung",
    appearanceDescription: "Wähle aus, wie Konfidence auf diesem Gerät dargestellt wird.",
    avatarFor: "Avatar für {name}",
    close: "Einstellungen schließen",
    description: "Verwalte dein Profil, das Anwendungsdesign und die Sprache.",
    language: "Sprache",
    profile: "Profil",
    sections: "Einstellungsbereiche",
    theme: "Design",
    themes: {
      konfidence: { description: "Hell und fokussiert", label: "Konfidence" },
      "konfidence-dark": {
        description: "Arbeitsbereich für dunkle Umgebungen",
        label: "Konfidence Dunkel",
      },
      sap_horizon: { description: "Vertrautes Enterprise-Blau", label: "Horizon" },
    },
    title: "Einstellungen",
  },
  landscape: {
    canvas: "Phasenlandschaft",
    stages: "Auslieferungsphasen",
    title: "Auslieferungslandschaft",
  },
  vector: {
    columns: {
      deployment: "Vektorbereitstellung",
      landscape: "Landschaft",
      repository: "Repository",
      stage: "Phase",
      status: "Status",
      version: "Version",
    },
    empty: "Keine Vektorbereitstellungen entsprechen der Suche.",
    inventory: "Projektbestand",
    search: "Vektorbereitstellungen suchen",
    searchPlaceholder: "Vektorbereitstellungen suchen...",
    summary:
      "Versionierte Vektoren, die derzeit Projektphasen zugewiesen sind. {visible} von {total} werden angezeigt.",
    table: "Vektorbereitstellungen",
    title: "Vektorbereitstellungen",
  },
};
/* oxlint-enable sort-keys */

const setupI18n = (): void => {
  addMessages("en", en);
  addMessages("de", de);
  init({ fallbackLocale: DEFAULT_LOCALE, initialLocale: DEFAULT_LOCALE });
};

export { setupI18n };
