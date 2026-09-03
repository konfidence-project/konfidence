/**
 * Mobile sidebar drawer state. Transient — resetting on reload is desirable
 * because a phone user should not land in a page with the drawer already open.
 *
 * Desktop collapse-to-rail is intentionally deferred: the shell CSS ships the
 * expanded layout only, and the prototype's icon-only rail relied on UI5's
 * `NavigationLayout` which we no longer use.
 */
const drawer = $state({ open: false });

const openDrawer = (): void => {
  drawer.open = true;
};

const closeDrawer = (): void => {
  drawer.open = false;
};

const toggleDrawer = (): void => {
  drawer.open = !drawer.open;
};

export { drawer, openDrawer, closeDrawer, toggleDrawer };
