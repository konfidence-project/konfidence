# ExpandableGroup and SelectableRow

Import both from `@konfidence/design-system/components` (or the package root).

`ExpandableGroup` accepts `ariaLabel`, controlled `expanded`,
`onExpandedChange(boolean)` and a `header` snippet. The optional `summary` snippet
remains always visible when supplied. Omit it for a header-only collapsed group;
all rows then belong in `children`, including when there is only one row.
Optional `children` supplies the additional rows. Omit it when there is nothing
to expand: the header becomes static and there is no misleading trigger.
Optional `triggerId` provides a consumer focus-restoration target.

The header must contain noninteractive content. An outlined right/down chevron
communicates expansion. Skeleton Collapsible owns
accessible expansion and hidden-content behavior. Each group has independent,
consumer-owned expansion state; updates do not reset it.

`SelectableRow` accepts `ariaLabel`, `selected` (default false), `onselect`,
`children`, optional `actions` and optional selection-button `id`.
The selection button exposes `aria-pressed` and native Enter/Space interaction.
Optional `inset` aligns desktop content with the group header's chevron gutter
using internal padding. The entire row still receives hover/selection paint;
consumers must not simulate this indentation with external padding or margins.
Its children must be noninteractive phrasing content (use spans, not nested
buttons, links or headings). Controls belong in the separate `actions` snippet;
they cannot activate the sibling selection button or the group's header trigger.

Both modules use existing semantic tokens and wrap at narrow widths. Domain
objects, ordering, counts, filters, URL state and refresh behavior belong to
consumers. See Lists & rows in the design preview for a controlled example.
