# ExpandableGroup and SelectableRow

Import both from `@konfidence/design-system/components` (or the package root).
They provide single-level disclosure groups and selectable rows, including
independent actions. They do not implement a hierarchical ARIA tree.

## Group selection and expansion

```svelte
<script lang="ts">
  import { ExpandableGroup, SelectableRow } from "@konfidence/design-system/components";
  let expanded = $state(true);
  let selectedFile = $state<string | undefined>("package.json");
</script>

<ExpandableGroup ariaLabel="Changed files" label="Changed Files"
  bind:expanded bind:selectedValue={selectedFile}>
  {#snippet metadata()}3 files{/snippet}
  <SelectableRow ariaLabel="Select package.json" value="package.json" label="package.json">
    {#snippet metadata()}Modified{/snippet}
    {#snippet trailing()}JSON{/snippet}
  </SelectableRow>
  <SelectableRow ariaLabel="Select src/index.ts" value="src/index.ts" label="src/index.ts" />
  <SelectableRow ariaLabel="Unavailable file" value="unavailable" label="Unavailable file" disabled />
</ExpandableGroup>
```

`expanded` defaults to false and can be bound, left local to the component, or
controlled with `onExpandedChange(boolean)`. `selectedValue` can also be bound.
Rows with a unique `value` share the nearest group's single selection. Selecting
another row clears the previous selection within that group; other groups are
unaffected. Selecting the same value again does not fire `onValueChange` or the
row's optional `onselect` callback again. Collapsing retains selection.
`onValueChange(value)` fires for user changes; initial and external assignments
remain silent. The application owns the data and should clear `selectedValue`
if the selected item is removed.

Rows without a `value` keep the existing application-controlled interface:
`selected` (default false) and `onselect`. They can be used outside a group and
support application-defined single or multiple selection. When a row has a
`value` inside a group, the group owns its selection and `selected` is ignored.

## Content and independent actions

Use `label` for a standard title and optional `metadata` for the secondary line.
Every row can use a single line, secondary text or multiple lines of custom
content. Titles and metadata wrap naturally without truncation or a line limit;
the row grows with its content above the 64px minimum height. These are content
variants of the same component, with identical selection behavior. Always-visible
summary rows and rows in static groups support the same content options.
Custom group `header` or row `children` snippets remain supported and replace
the standard title. Omitted titles fall back to `ariaLabel`. Optional row
`trailing` content sits on the right inside the selection button; clicking it
selects the row. All content inside these buttons must be noninteractive
phrasing content, such as spans and text.

Put independent buttons and links in the row's separate `actions` snippet.
They do not select the row or expand its group. `disabled` disables the native
selection button, removes it from the tab order and dims its content. Independent
actions retain their own enabled/disabled state.

The optional group `summary` stays visible while collapsed. Omit it for a
header-only collapsed group, placing all rows in `children`. Omit `children`
when there is nothing to expand: the header becomes static without a misleading
trigger. Each group's expansion remains independent. Skeleton Collapsible owns
accessible expansion and hidden-content behavior.

Optional `triggerId` (group) and `id` (row selection button) provide focus
restoration targets. Both components accept `class` for consumer layout.
`inset` aligns desktop row content with the header's chevron gutter using
internal padding; the selection marker and hover surface still span the row.

## Visuals and accessibility

The standard layout uses existing design tokens: 14px container corners,
13px text, a 16px SAP chevron, 16px/12px header padding, 16px row padding and
a 64px minimum selection-button height. Selected rows use the semantic
`selection-bg` and a 3px full-height amber gradient marker. The card surface
stays consistent when expanded or collapsed. Actions wrap at narrow widths.

Native buttons provide Tab, Enter and Space interaction. Selection exposes
`aria-pressed`; disclosure exposes `aria-expanded`. Collapsed rows and disabled
selection buttons leave the tab order. Reduced motion disables transitions.
Light/dark behavior follows the existing developer tokens. Table and StageCard
retain their existing 20px card corners; these smaller disclosure groups use the
existing 14px container radius from the Designer DS reference.

See **Lists & rows** in the design preview for single-line and multiple-line rows,
an always-visible row, independent actions and disabled rows.
