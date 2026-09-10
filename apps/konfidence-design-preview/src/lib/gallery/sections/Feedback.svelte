<script lang="ts">
    import NotYetImplemented from "../NotYetImplemented.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="feedback"
    title="Feedback states"
    subtitle="Toasts, empty state, skeleton loaders."
    status="placeholder"
>
    <NotYetImplemented
        note="Skeleton v5 provides the underlying Toast primitive. Konfidence has not yet published an opinionated wrapper."
        planned={["<Toast>", "<EmptyState>", "<SkeletonBox> / <SkeletonText>"]}
    />
</Section>
