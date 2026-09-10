<script lang="ts">
    import { Button, IconButton } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="buttons"
    title="Buttons"
    subtitle="Real <Button> and <IconButton> components from @konfidence/design-system."
    status="implemented"
>
    <div class="stack">
        <Sample title="Variants" description="Primary should appear at most once per screen.">
            <Button variant="primary">Primary</Button>
            <Button variant="secondary">Secondary</Button>
            <Button variant="ghost">Ghost</Button>
            <Button variant="danger">Danger</Button>
        </Sample>

        <Sample title="As anchor" description="`href` prop renders <a> with the same styles.">
            <Button variant="primary" href="#buttons">Learn more</Button>
            <Button variant="secondary" href="#buttons">Docs</Button>
        </Sample>

        <Sample title="Disabled">
            <Button variant="primary" disabled>Primary</Button>
            <Button variant="secondary" disabled>Secondary</Button>
            <Button variant="ghost" disabled>Ghost</Button>
            <Button variant="danger" disabled>Danger</Button>
        </Sample>

        <Sample
            title="Icon buttons"
            description="`<IconButton>` requires an aria-label; supports a numeric badge overlay."
        >
            <IconButton icon="add" ariaLabel="Add" />
            <IconButton icon="edit" ariaLabel="Edit" />
            <IconButton icon="filter" ariaLabel="Filter" />
            <IconButton icon="bell" ariaLabel="Notifications" badge={3} />
            <IconButton icon="bell" ariaLabel="Notifications" badge={128} />
        </Sample>
    </div>
</Section>

<style>
    .stack {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }
</style>
