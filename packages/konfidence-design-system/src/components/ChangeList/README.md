# ChangeList

A read-only diff list ported from the Designer DS. Green Added, amber Changed
and red Removed rows combine a 3px left marker, a compact visible status label,
a monospace name and optional previous/new values. Status is supplied by the
consumer; the component does not calculate diffs or perform actions.

```svelte
<script lang="ts">
  import { ChangeList, type ChangeListItem } from "@konfidence/design-system";

  const changes: ChangeListItem[] = [
    { id: "sdk", state: "added", name: "analytics-sdk", newValue: "1.2.0" },
    { id: "payment", state: "changed", name: "payment-svc", oldValue: "2.13.1", newValue: "2.14.0" },
    { id: "legacy", state: "removed", name: "legacy-adapter", oldValue: "0.9.4" },
    { id: "search", state: "changed", label: "Enabled", name: "new-search", oldValue: false, newValue: true },
  ];
</script>

<ChangeList items={changes} ariaLabel="Artifact and feature toggle changes"
  emptyLabel="No changes in this promotion." />
```

| Prop         | Default       | Meaning                                                                    |
| ------------ | ------------- | -------------------------------------------------------------------------- |
| `items`      | Required      | Ordered readonly array of `ChangeListItem`; use unique stable `id` values. |
| `ariaLabel`  | `Changes`     | Accessible name of the nonempty list.                                      |
| `emptyLabel` | `No changes.` | Inline empty-section text when `items` is empty.                           |
| `class`      | —             | Extra classes on the outer container.                                      |

Items require `id`, `name` and `state` (`added`, `changed`, `removed`). `label`
overrides the visible default state label, e.g. Enabled or Disabled. `group`
adds an uppercase heading at the start of a consecutive group. The list retains
input order and does not sort or merge groups.

`oldValue` and `newValue` accept strings or booleans. Added shows only the new
value; Removed shows only the old value; Changed shows the supplied values on
either side of an arrow. Inapplicable values are ignored. `undefined` means
omitted, while `false` and the empty string remain valid values. Omit both to
show only the status and name. Screen readers receive Previous value/New value
labels; the decorative arrow is hidden from them.

Boolean values use the exported `TogglePill` component:

```svelte
<TogglePill on={true} />
<TogglePill on={false} />
```

`TogglePill` accepts required boolean `on` and optional `class`. It displays
On/Off as text, with a green/neutral tone; it has no interactive switch role.

Names truncate visually and expose the full text in a native title and to
assistive technology. Values wrap on a second row below 380px **container**
width, including inside desktop side panels. Longer values wrap rather than
causing horizontal overflow. The green/red reference colors and compact label
geometry are local to these components; existing shared tokens are unchanged.
The On pill uses the existing theme-aware healthy text token for dark mode.

Preview: `/#change-list`, including all states, toggles, grouping, omitted
values, long content, empty sections and standalone pills. No selection,
row actions, loading or error states are defined in this first version.
