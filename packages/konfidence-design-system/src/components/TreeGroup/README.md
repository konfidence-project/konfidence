# TreeGroup and TreeRow

Single-level disclosure groups with an independent, persistent single selection.
Ported from the Designer DS Tree-Group reference (`components.css`, DS v1.3.9).

```svelte
<script lang="ts">
  import { TreeGroup, TreeRow } from "@konfidence/design-system";
  let open = $state(true);
  let selected = $state<string>();
</script>

<TreeGroup label="Changed Files" bind:open bind:selectedValue={selected}>
  {#snippet metadata()}2 files{/snippet}
  <TreeRow value="index" label="src/index.ts">
    {#snippet metadata()}modified{/snippet}
    {#snippet trailing()}TypeScript{/snippet}
  </TreeRow>
  <TreeRow value="readme" label="README.md" />
</TreeGroup>
```

`TreeGroup` accepts `label`, optional `metadata`, `children`, `open` (default
`false`), `selectedValue`, `onOpenChange`, `onValueChange`, and `class`. Both state
props are bindable. Callbacks fire for user changes, not initial or external
assignments. Selecting the current row again leaves it selected. Each group owns
its selection; collapsing does not reset it. The consumer owns data updates and
should clear `selectedValue` if its corresponding row is removed.

`TreeRow` must be inside a TreeGroup. It accepts a unique string `value`, `label`,
optional `metadata` and `trailing` snippets, `disabled`, and `class`. Snippets
render inside native buttons: use non-interactive phrasing content, such as
spans, text, or icons, rather than nested links, buttons, or inputs.

The header is a disclosure button with `aria-expanded` and `aria-controls`;
rows are list items containing buttons with `aria-pressed`. Tab, Enter and Space
use native button behavior. This is not a hierarchical ARIA tree: it deliberately
does not imply arrow-key tree navigation. Collapsed rows leave the tab order.

Visuals preserve the reference's 14px corners, 13px labels, 16px chevron, header
and row spacing, 64px minimum row height, and amber selection treatment. The
3px selection marker spans the full row height, with square ends.
There is no distinct compact layout in the source Tree-Group rules. Styling uses
existing semantic utilities/tokens and honors reduced motion. The surface uses
`surface-card`, so dark mode follows the developer theme; the Designer DS's
`card-bg` remains white in dark mode. No global CSS or theme persistence is added.
