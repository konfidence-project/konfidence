# ExpandableGroup and SelectableRow

Import both from `@konfidence/design-system/components`. They provide single-level
disclosure and selection, not a hierarchical ARIA tree.

```svelte
<script lang="ts">
  import { ExpandableGroup, SelectableRow } from "@konfidence/design-system/components";
  let expanded = $state(false);
  let selectedFile = $state<string>();
</script>

<ExpandableGroup ariaLabel="Changed files" bind:expanded bind:selectedValue={selectedFile}>
  {#snippet header()}
    <div>Changed files</div>
    <button onclick={(event) => {
      event.stopPropagation();
      expanded = true;
      selectedFile = "package.json";
    }}>Show package.json</button>
  {/snippet}
  <SelectableRow ariaLabel="Select package.json" value="package.json" id="package-row" inset>
    <span class="block font-semibold">package.json</span>
    <span class="block">Modified · JSON</span>
    {#snippet actions()}<button>Inspect file</button>{/snippet}
  </SelectableRow>
</ExpandableGroup>
```

## Bindings and ownership

`expanded` defaults to false; `selectedValue` defaults to undefined. Both can stay
local or use ordinary Svelte bindings. Bind several groups to the same parent
value for shared selection. Values must be unique across that selection scope.
Selecting a row never toggles it off or expands its group; collapsing retains
selection. External assignment, mounting and unmounting do not replay user actions
or clear parent state. The application owns removal of selections absent from its data.

For application-owned state such as URL navigation, use a function binding:

```svelte
<ExpandableGroup ariaLabel="Files"
  bind:expanded={() => expansion.files ?? false, (open) => expansion.files = open}
  bind:selectedValue={() => selectedId, (id) => selectFile(id)}
  {header} {children} />
```

Normalize sparse expansion-map reads to false. A function setter is synchronous,
but asynchronous application navigation need not immediately update its getter.

## Content and events

`header` is a required free snippet outside the native chevron button. Clicking
noninteractive header content toggles disclosure. Independent header controls
must synchronously call `event.stopPropagation()` before any asynchronous work;
their handlers explicitly own any selection or expansion change. Consumers are
responsible for accessible semantics and keyboard operation of custom header controls.

Omit `children` when there is nothing to expand: the header becomes static, without
a chevron or toggle handler. An empty rendered snippet does not count as omission.
Skeleton Collapsible owns native disclosure behavior, content IDs and hidden content.

Rows require `value`, `ariaLabel` and a `children` snippet of noninteractive phrasing
content (text and spans, not nested controls). A row outside a group throws a
descriptive programming error. Put independent controls in `actions`; they are
siblings of the selection button and own their enabled/disabled states.

Optional `triggerId` and row `id` identify focus-restoration targets. Static headers
support programmatic focus without becoming tab stops. `inset` aligns desktop row
content with the chevron gutter.

## Visuals and accessibility

Groups retain 14px corners and a stable card surface. Rows wrap naturally above a
64px minimum height; actions wrap independently. Selection uses `aria-pressed`,
a semantic selected background (also on hover) and a full-height 3px amber marker.
The named native chevron supports Enter and Space, exposes `aria-expanded` and
`aria-controls`, and keeps collapsed rows out of keyboard navigation. Focus rings,
reduced motion and light/dark tokens remain shared with the design system.

See **Lists & rows** in the preview for local/shared selection, free interactive
headers, static groups and independent actions.
