<script lang="ts">
    import { Breadcrumbs } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";

    let { name = "Breadcrumbs" }: { name?: string } = $props();
</script>

<Section
    id="navigation"
    title={name}
    subtitle="Navigation trails with a noninteractive current-page marker."
>
    <div class="stack flex flex-col gap-5">
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

    </div>
</Section>
