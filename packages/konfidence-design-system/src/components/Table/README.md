# Table row interaction

Selectable table rows use the same interaction colors as selectable list rows:

- Selection: the existing selection background and a 3px full-height amber gradient marker at the left edge. The marker does not change cell spacing or column widths.
- Hover: a neutral subtle surface for unselected interactive rows. Selected rows retain their selection background and marker on hover.
- Keyboard focus: a separate 2px inset amber outline, visible whether or not the row is selected.
- Reduced motion: background transitions are disabled when requested.

Use TableRow with onselect to enable click, Enter and Space selection. The consumer controls selected; aria-selected exposes that state. Static rows do not show interactive hover or focus affordances.

Table typography, cell spacing, sorting, headers, card radius and shadow remain unchanged.
