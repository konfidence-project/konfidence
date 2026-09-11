<script lang="ts">
    import SearchInput from "../SearchInput.svelte";

    interface Props {
        value?: string;
        placeholder?: string;
    }
    let {
        value = $bindable(""),
        placeholder = "Search artifact, version, stage\u2026",
    }: Props = $props();
</script>

<div style="width: 360px; padding: 16px; background: var(--surface-canvas);">
    <SearchInput bind:value {placeholder} aria-label="Search artifact deployments" />
</div>
