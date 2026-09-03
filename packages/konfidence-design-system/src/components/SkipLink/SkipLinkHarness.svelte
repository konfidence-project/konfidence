<script lang="ts">
    import SkipLink from "./SkipLink.svelte";

    interface Props {
        target: string;
        label: string;
    }

    let { target, label }: Props = $props();
</script>

<SkipLink {target}>{label}</SkipLink>
