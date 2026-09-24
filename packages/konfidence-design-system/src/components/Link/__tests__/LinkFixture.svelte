<script lang="ts">
    import Link from "../Link.svelte";

    interface Props {
        href?: string;
        disabled?: boolean;
        class?: string;
        "aria-label"?: string;
        label: string;
        onclick?: (event: MouseEvent) => void;
    }

    let { label, ...rest }: Props = $props();
</script>

<Link {...rest}>{label}</Link>