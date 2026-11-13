<script lang="ts">
    import ActivityFeed from "./ActivityFeed.svelte";
    import Atoms from "./Atoms.svelte";
    import Charts from "./Charts.svelte";
    import LogsAndCode from "./LogsAndCode.svelte";
    import MessageStrips from "./MessageStrips.svelte";
    import Misc from "./Misc.svelte";
    import Tags from "./Tags.svelte";
    import NotYetImplemented from "../NotYetImplemented.svelte";

    let { name = "Roadmap" }: { name?: string } = $props();
</script>

<section aria-label={name}>
<NotYetImplemented planned={["<KpiTile>", "<Phases>", "<DiffRow>", "<Timeline>", "<KeyValue>", "<TextInput>", "<Checkbox>", "<RadioGroup>", "<Switch>", "<Tabs>", "<Segmented>", "<Filterbar>", "<Toast>", "<SkeletonBox>", "<ObjectHeader>", "<Promotion>", "<PipelineTimeline>", "<Dialog>", "<MasterDetail>", "<ActivityRow>", "<FilterCard>"]} />
<ActivityFeed /><Atoms /><Charts /><LogsAndCode /><MessageStrips /><Misc /><Tags />
</section>
