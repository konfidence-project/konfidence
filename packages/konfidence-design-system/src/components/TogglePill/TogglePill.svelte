<script lang="ts">
    interface Props {
        /** Read-only boolean status; this is not an interactive switch. */
        on: boolean;
        class?: string;
    }
    let { on, class: className }: Props = $props();
</script>

<span class={["toggle-pill inline-flex shrink-0 items-center whitespace-nowrap rounded-full border px-[7px] py-px text-meta font-semibold", on ? "toggle-pill--on border-status-healthy-solid/30 bg-status-healthy-bg text-status-healthy-fg" : "border-outline-subtle bg-surface-sunken text-content-tertiary", className]}>{on ? "On" : "Off"}</span>
