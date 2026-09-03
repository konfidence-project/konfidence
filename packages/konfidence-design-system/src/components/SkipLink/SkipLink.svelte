<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /**
         * The id of the element focus should jump to when the link is
         * activated. Pass the same id you set on the `<main>` element
         * inside `AppShell`.
         */
        target: string;
        /** Extra class names for layout tweaks. */
        class?: string;
        /** Link label. Required — a skip link without visible text is useless. */
        children: Snippet;
    }

    let { target, class: className, children }: Props = $props();

    const composedClass = $derived(className ? `skip-link ${className}` : "skip-link");
</script>

<a class={composedClass} href={`#${target}`}>
    {@render children()}
</a>
