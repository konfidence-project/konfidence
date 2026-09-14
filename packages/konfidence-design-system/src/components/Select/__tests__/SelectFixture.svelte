<script lang="ts">
    import Select from "../Select.svelte";

    interface Props {
        value?: string;
        disabled?: boolean;
    }
    let { value = $bindable(""), disabled = false }: Props = $props();
</script>

<div style="width: 260px; padding: 16px; background: var(--surface-canvas);">
    <Select bind:value {disabled} aria-label="Filter by status">
        <option value="">All statuses</option>
        <option value="deployed">Deployed</option>
        <option value="fetched">Fetched</option>
    </Select>
</div>
