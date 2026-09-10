/**
 * Whether a resolved destination href matches the current pathname.
 * Matching is exact or "path is nested under href", so
 * `/projects/p/landscape/detail` still highlights the Landscape
 * destination. A trailing `/` guard prevents `/projects/p/landscape-x`
 * from matching `/projects/p/landscape`.
 *
 * Kept next to `NavItem` because it is the intended default when the
 * consumer wants automatic active-state tracking off a router-provided
 * pathname.
 */
const isActive = (pathname: string, href: string): boolean =>
  pathname === href || pathname.startsWith(`${href}/`);

export { isActive };
