<script lang="ts">
    import { setContext } from "svelte";
    import { entryId } from "./catalog.ts";
    import type { CatalogEntry } from "./catalog.ts";
    import { CATALOG_ENTRY } from "./catalog-entry-context.ts";
    import type { CurrentCatalogEntry } from "./catalog-entry-context.ts";

    let { entry, hidden = false }: { entry: CatalogEntry; hidden?: boolean } = $props();
    setContext<CurrentCatalogEntry>(CATALOG_ENTRY, () => entry);
</script>

<section id={entryId(entry)} aria-label={entry.name} data-catalog-status={entry.status} {hidden} class={["scroll-mt-24", hidden && "hidden"]}>
    <entry.demo name={entry.name} />
</section>
