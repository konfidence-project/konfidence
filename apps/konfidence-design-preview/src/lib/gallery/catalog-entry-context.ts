import type { CatalogEntry } from "./catalog.ts";

// Metadata belongs to the keyed catalog item, not to its reusable demo or the DS.
const CATALOG_ENTRY = Symbol("design-preview-catalog-entry");
type CurrentCatalogEntry = () => CatalogEntry;

export { CATALOG_ENTRY };
export type { CurrentCatalogEntry };
