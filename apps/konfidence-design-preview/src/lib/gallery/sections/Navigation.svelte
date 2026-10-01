<script lang="ts">
    import { Breadcrumbs } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="navigation"
    title="Navigation primitives"
    subtitle="Real <Breadcrumbs> from @konfidence/design-system, plus placeholders for the remaining in-page navigation primitives (tabs, segmented, filterbar)."
    status="partial"
>
    <div class="stack">
        <Sample
            title="Breadcrumbs — three crumbs"
            description="Last crumb is always non-interactive with aria-current=page; intermediate crumbs may omit `href` to render as plain text."
        >
            <Breadcrumbs
                items={[
                    { href: "#navigation", label: "Projects" },
                    { href: "#navigation", label: "checkout" },
                    { label: "Artifact deployments" },
                ]}
            />
        </Sample>
        <Sample title="Breadcrumbs — two crumbs">
            <Breadcrumbs
                items={[
                    { href: "#navigation", label: "Projects" },
                    { label: "checkout" },
                ]}
            />
        </Sample>
        <Sample
            title="Breadcrumbs — single current"
            description="With only one item the trail collapses to a lone current-page marker."
        >
            <Breadcrumbs items={[{ label: "Projects" }]} />
        </Sample>

        <NotYetImplemented
            note="In-page navigation primitives beyond breadcrumbs are still pending. Shell navigation (topbar, sidebar, NavGroup, NavItem) is already shipped and demonstrated in the App shell section below."
            planned={[
                "<Tabs> (with overflow menu)",
                "<Segmented> / <MultiButton>",
                "<Filterbar>",
            ]}
        />
    </div>
</Section>

<style>
    .stack {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }
</style>
