import { Menu as SkMenu } from "@skeletonlabs/skeleton-svelte";
import MenuContent from "./MenuContent.svelte";
import MenuHeader from "./MenuHeader.svelte";
import MenuItem from "./MenuItem.svelte";
import MenuLabel from "./MenuLabel.svelte";
import MenuPositioner from "./MenuPositioner.svelte";
import MenuRoot from "./MenuRoot.svelte";
import MenuSeparator from "./MenuSeparator.svelte";

/**
 * Konfidence menu — Skeleton `Menu.*` slots with the design-system
 * `.menu*` class vocabulary pre-applied.
 *
 * Composition rules:
 * - `Menu` renders Skeleton's state provider (via `MenuRoot`). Pass
 *   `positioning`, `onSelect`, etc. through as usual.
 * - `Menu.Trigger` is re-exported from Skeleton unchanged: consumers
 *   apply their own trigger class (`.avatar`, `.project-switch`) because
 *   trigger visuals are context-specific.
 * - `Menu.ItemGroup` is re-exported unchanged from Skeleton for
 *   consumers who want to group options.
 * - `Menu.Positioner` adds `.menu-positioner` so the panel stacks
 *   above the mobile drawer (Zag emits an inline `z-index`; the class
 *   uses `!important` to win).
 * - `Menu.Content` adds `.menu` (+ `menu--sm/lg`, `menu--header`).
 * - `Menu.Item` adds `.menu__item` (+ `--danger`, `--active`).
 * - `Menu.Separator` adds `.menu__sep`.
 * - `Menu.Label` (Skeleton's `ItemGroupLabel`) adds `.menu__label`.
 * - `Menu.Header` is a non-Skeleton block that renders the
 *   `.menu__header` chrome (avatar + name + mail) inside a
 *   `menu--header` content.
 */
const Menu = Object.assign(MenuRoot, {
  Content: MenuContent,
  Header: MenuHeader,
  Item: MenuItem,
  ItemGroup: SkMenu.ItemGroup,
  Label: MenuLabel,
  Positioner: MenuPositioner,
  Separator: MenuSeparator,
  Trigger: SkMenu.Trigger,
});

export { Menu };
