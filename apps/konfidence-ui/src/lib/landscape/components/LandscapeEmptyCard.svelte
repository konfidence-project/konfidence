<script lang="ts">
    import { StageCardPlaceholder } from "@konfidence/design-system/components";

    interface Props {
        landscapeName: string;
    }
    let { landscapeName }: Props = $props();
</script>

<!-- A landscape with no stages still appears on the canvas, laid out with the
     same card frame as a populated stage via the design-system placeholder. -->
<StageCardPlaceholder
    title={landscapeName}
    message="No stages yet"
    ariaLabel={`Landscape ${landscapeName} has no stages yet`}
/>
