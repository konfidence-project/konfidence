<script lang="ts">
    import logo from "../../assets/logo.svg?raw";

    let { class: className }: { class?: string } = $props();
</script>

<span
    role="img"
    aria-label="Konfidence"
    class={["block max-w-full text-black dark:text-white [&_svg]:h-auto [&_svg]:w-full [&_path]:fill-current", className]}
>
    <!-- eslint-disable-next-line svelte/no-at-html-tags -- Repository-owned SVG imported at build time. -->
    {@html logo}
</span>
