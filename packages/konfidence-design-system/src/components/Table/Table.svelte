<script lang="ts">
    import type { HTMLAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";

    interface Props extends Omit<HTMLAttributes<HTMLTableElement>, "class" | "children"> {
        class?: string;
        header: Snippet;
        body: Snippet;
        caption?: string;
    }

    let { class: className, header, body, caption, ...rest }: Props = $props();

    const tableClass = $derived(
        [
            "w-full border-collapse text-[length:var(--text-sm)] text-[color:var(--text-primary)]",
            "[font-variant-numeric:tabular-nums]",
            className,
        ]
            .filter(Boolean)
            .join(" "),
    );
</script>

<div class="overflow-hidden rounded-[var(--card-radius)] border border-[color:var(--border-subtle)] bg-[color:var(--surface-card)] shadow-[var(--card-shadow)]">
    <div class="max-h-[min(65vh,45rem)] overflow-auto">
        <table class={tableClass} {...rest}>
            {#if caption}
                <caption class="sr-only">{caption}</caption>
            {/if}
            <thead>{@render header()}</thead>
            <tbody>{@render body()}</tbody>
        </table>
    </div>
</div>

<style>
    .sr-only {
        position: absolute;
        width: 1px;
        height: 1px;
        padding: 0;
        margin: -1px;
        overflow: hidden;
        clip: rect(0, 0, 0, 0);
        white-space: nowrap;
        border: 0;
    }
</style>
