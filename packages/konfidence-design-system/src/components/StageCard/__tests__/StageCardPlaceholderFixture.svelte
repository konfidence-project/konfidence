<script lang="ts">
    import StageCardPlaceholder from "../StageCardPlaceholder.svelte";

    interface Props {
        title: string;
        landscapeName?: string;
        message?: string;
        ariaLabel?: string;
    }

    let props: Props = $props();
</script>

<StageCardPlaceholder {...props} />
