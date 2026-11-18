import type { Component } from "svelte";
import type * as Components from "@konfidence/design-system/components";
import Buttons from "./sections/Buttons.svelte";
import Colors from "./sections/Colors.svelte";
import DataBlocks from "./sections/DataBlocks.svelte";
import Feedback from "./sections/Feedback.svelte";
import Forms from "./sections/Forms.svelte";
import Hero from "./sections/Hero.svelte";
import Icons from "./sections/Icons.svelte";
import Lists from "./sections/Lists.svelte";
import Navigation from "./sections/Navigation.svelte";
import Overlays from "./sections/Overlays.svelte";
import Pipeline from "./sections/Pipeline.svelte";
import Planned from "./sections/Planned.svelte";
import Primitives from "./sections/Primitives.svelte";
import Roadmap from "./sections/Roadmap.svelte";
import Shell from "./sections/Shell.svelte";
import ShellPatterns from "./sections/ShellPatterns.svelte";
import Signature from "./sections/Signature.svelte";
import StatusBadges from "./sections/StatusBadges.svelte";
import Typography from "./sections/Typography.svelte";

interface CatalogEntry {
  name: string;
  kind: "Component" | "Foundation" | "Roadmap";
  demo: Component<{ name?: string }>;
  keywords: string;
  readme?: string;
  status: "implemented" | "planned";
}

// The package's public exports own component names; only preview presentation
// (examples, search aliases and shared documentation) belongs here.
const componentDemos = {
  AppShell: Shell,
  Avatar: ShellPatterns,
  AvatarGroup: ShellPatterns,
  BrandLogo: Primitives,
  Brandbar: Primitives,
  Breadcrumbs: Navigation,
  Button: Buttons,
  ColorModeSelect: Primitives,
  EmptyState: Feedback,
  ExpandableGroup: Lists,
  IconButton: Buttons,
  Link: Primitives,
  Menu: ShellPatterns,
  NavGroup: Shell,
  NavItem: Shell,
  OrbitLoader: Primitives,
  PageHeader: Hero,
  ProjectSelection: ShellPatterns,
  SearchInput: Forms,
  Select: Forms,
  SelectableRow: Lists,
  SidePanel: Overlays,
  Sidebar: Shell,
  StageCard: Pipeline,
  StageCardPlaceholder: Primitives,
  StagePhase: Pipeline,
  StatusBadge: StatusBadges,
  Table: DataBlocks,
  TableCell: DataBlocks,
  TableHeaderCell: DataBlocks,
  TableRow: DataBlocks,
  TopBar: Shell,
} satisfies Record<Exclude<keyof typeof Components, "isActive">, Component<{ name?: string }>>;

const aliases: Partial<Record<keyof typeof componentDemos, string>> = {
  AppShell: "application layout",
  Breadcrumbs: "navigation trail",
  ColorModeSelect: "theme light dark system",
  EmptyState: "feedback error placeholder",
  ExpandableGroup: "lists rows disclosure collapsible",
  Menu: "dropdown user account",
  OrbitLoader: "loading spinner progress",
  ProjectSelection: "project switcher",
  SearchInput: "forms filter",
  Select: "forms dropdown",
  SelectableRow: "lists rows selection",
  SidePanel: "drawer dialog overlay details",
  StageCard: "pipeline deployment",
  StagePhase: "pipeline progress",
  Table: "data sortable selectable",
};

const entries = [
  ...Object.entries(componentDemos).map(([name, demo]) => ({
    demo,
    keywords: aliases[name as keyof typeof componentDemos] ?? "",
    kind: "Component" as const,
    name,
    readme: name === "SelectableRow" ? "ExpandableGroup" : name,
    status: "implemented" as const,
  })),
  {
    demo: Colors,
    keywords: "palette tokens gradients",
    kind: "Foundation",
    name: "Colors",
    status: "implemented",
  },
  {
    demo: Icons,
    keywords: "SAP symbols",
    kind: "Foundation",
    name: "Icons",
    status: "implemented",
  },
  {
    demo: Signature,
    keywords: "design language amber teal",
    kind: "Foundation",
    name: "Signature",
    status: "implemented",
  },
  {
    demo: Typography,
    keywords: "fonts spacing radii tokens",
    kind: "Foundation",
    name: "Typography",
    status: "implemented",
  },
  {
    demo: Roadmap,
    keywords: "pending planned components",
    kind: "Roadmap",
    name: "Roadmap",
    status: "planned",
  },
  ...[
    "ActivityFeed",
    "ActivityRow",
    "AlertTile",
    "Checkbox",
    "CodeBlock",
    "Dialog",
    "DiffRow",
    "Filterbar",
    "FilterCard",
    "KeyValue",
    "KpiTile",
    "LogTable",
    "MasterDetail",
    "MessageStrip",
    "ObjectHeader",
    "Phases",
    "PipelineTimeline",
    "Promotion",
    "RadioGroup",
    "Segmented",
    "SkeletonBox",
    "Switch",
    "Tabs",
    "Tag",
    "TextInput",
    "Timeline",
    "Toast",
  ].map((name) => ({
    demo: Planned,
    keywords: "pending roadmap",
    kind: "Component" as const,
    name,
    status: "planned" as const,
  })),
] satisfies CatalogEntry[];
const CATALOG: readonly CatalogEntry[] = entries.toSorted((left, right) =>
  left.name.localeCompare(right.name),
);

const legacySections: Readonly<Record<string, string>> = {
  "activity-feed": "Roadmap",
  buttons: "Button",
  charts: "Roadmap",
  "data-blocks": "Table",
  feedback: "EmptyState",
  forms: "SearchInput",
  "foundation-atoms": "Roadmap",
  "foundation-colors": "Colors",
  "foundation-tags": "Roadmap",
  "foundation-typography": "Typography",
  hero: "PageHeader",
  icons: "Icons",
  lists: "ExpandableGroup",
  "logs-code": "Roadmap",
  "message-strips": "Roadmap",
  misc: "Roadmap",
  navigation: "Breadcrumbs",
  overlays: "SidePanel",
  pipeline: "StageCard",
  shell: "AppShell",
  "shell-patterns": "ProjectSelection",
  signature: "Signature",
  "status-badges": "StatusBadge",
};

const selectedEntry = (url: {
  hash: string;
  searchParams: Pick<URLSearchParams, "get">;
}): CatalogEntry | undefined => {
  const hash = url.hash.slice(1);
  const name = hash.startsWith("component-")
    ? hash.slice("component-".length)
    : (legacySections[hash] ?? url.searchParams.get("component"));
  return CATALOG.find((entry) => entry.name === name);
};

const searchCatalog = (query: string): readonly CatalogEntry[] => {
  const terms = query.trim().toLowerCase().split(/\s+/);
  const matches = CATALOG.filter((entry) => {
    const text = `${entry.name} ${entry.kind} ${entry.status} ${entry.keywords}`.toLowerCase();
    return terms.every((term) => text.includes(term));
  });
  const normalized = query.trim().toLowerCase();
  const rank = (entry: CatalogEntry): number => {
    const name = entry.name.toLowerCase();
    if (name === normalized) {
      return 0;
    }
    if (name.startsWith(normalized)) {
      return 1;
    }
    return 2;
  };
  return matches.toSorted((left, right) => rank(left) - rank(right));
};

const entryId = (entry: CatalogEntry): string => `component-${entry.name}`;

export { CATALOG, entryId, searchCatalog, selectedEntry };
export type { CatalogEntry };
