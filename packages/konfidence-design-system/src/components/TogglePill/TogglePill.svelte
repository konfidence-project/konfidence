<script lang="ts">
    interface Props {
        /** Read-only boolean status; this is not an interactive switch. */
        on: boolean;
        class?: string;
    }
    let { on, class: className }: Props = $props();
</script>

<span class={["toggle-pill inline-flex shrink-0 items-center whitespace-nowrap rounded-full border px-[7px] py-px text-meta font-semibold", on ? "toggle-pill--on text-status-healthy-fg" : "border-outline-subtle bg-surface-sunken text-content-tertiary", className]}>{on ? "On" : "Off"}</span>

<style>
    .toggle-pill--on {
        background: color-mix(in srgb, #19a765 15%, transparent);
        border-color: color-mix(in srgb, #19a765 30%, transparent);
    }
</style>
