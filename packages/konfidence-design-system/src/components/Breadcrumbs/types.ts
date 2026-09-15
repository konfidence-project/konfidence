interface BreadcrumbItem {
  /** Text shown for this crumb. Required. */
  label: string;
  /**
   * Destination when the crumb is a link. When omitted the crumb
   * renders as a plain `<span>` — used for the current (last) crumb
   * or for placeholder crumbs whose destination does not exist yet.
   */
  href?: string;
}

export type { BreadcrumbItem };
