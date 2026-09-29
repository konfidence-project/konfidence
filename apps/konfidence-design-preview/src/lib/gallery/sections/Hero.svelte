<script lang="ts">
    import {
        Breadcrumbs,
        Button,
        IconButton,
        PageHeader,
    } from "@konfidence/design-system/components";
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="hero"
    title="Page header"
    subtitle="Real <PageHeader> from @konfidence/design-system — title, optional description, optional eyebrow (breadcrumbs, back link), optional right-aligned actions. Higher-density object headers with KPIs still pending."
    status="partial"
>
    <div class="stack flex flex-col gap-5">
        <Sample title="Title only">
            <PageHeader class="page-header-demo w-full" title="Projects" />
        </Sample>

        <Sample title="Title + description">
            <PageHeader
                class="page-header-demo w-full"
                title="Artifact deployments"
                description="Inspect the artifact deployments backing this project's vectors, stages, and landscapes."
            />
        </Sample>

        <Sample
            title="With eyebrow (breadcrumbs)"
            description="Eyebrow slot renders above the title — typically breadcrumbs or a back link."
        >
            <PageHeader class="page-header-demo w-full" title="checkout">
                {#snippet eyebrow()}
                    <Breadcrumbs
                        items={[
                            { href: "#hero", label: "Projects" },
                            { label: "checkout" },
                        ]}
                    />
                {/snippet}
            </PageHeader>
        </Sample>

        <Sample
            title="With actions"
            description="Actions slot renders right-aligned on the title row; wraps to a new row on narrow viewports."
        >
            <PageHeader
                class="page-header-demo w-full"
                title="Landscapes"
                description="Every landscape the current project can deploy into."
            >
                {#snippet actions()}
                    <Button variant="secondary" icon="filter">Filter</Button>
                    <Button variant="primary" icon="add">New landscape</Button>
                {/snippet}
            </PageHeader>
        </Sample>

        <Sample title="All slots">
            <PageHeader
                class="page-header-demo w-full"
                title="prod"
                description="EU · 3 stages · deployed 12m ago"
            >
                {#snippet eyebrow()}
                    <Breadcrumbs
                        items={[
                            { href: "#hero", label: "Projects" },
                            { href: "#hero", label: "checkout" },
                            { href: "#hero", label: "Landscapes" },
                            { label: "prod" },
                        ]}
                    />
                {/snippet}
                {#snippet actions()}
                    <IconButton icon="refresh" ariaLabel="Refresh" />
                    <Button variant="secondary" icon="download">Export</Button>
                    <Button variant="primary" icon="activate">Promote</Button>
                {/snippet}
            </PageHeader>
        </Sample>

        <NotYetImplemented
            note="A high-density `<ObjectHeader>` variant (KPI cluster, status ribbon, secondary metadata rows) is still pending."
            planned={["<ObjectHeader> (KPIs + tags + status ribbon)"]}
        />
    </div>
</Section>
