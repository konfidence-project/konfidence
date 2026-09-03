<script lang="ts">
    import Button from "../components/Button.svelte";

    interface Props {
        variant?: "primary" | "secondary" | "ghost" | "danger";
        href?: string;
        disabled?: boolean;
        "aria-label"?: string;
        label: string;
        onclick?: () => void;
    }

    let { label, ...rest }: Props = $props();
</script>

<Button {...rest}>{label}</Button>
