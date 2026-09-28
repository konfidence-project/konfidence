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
</script>

<div class="overflow-hidden rounded-[var(--card-radius)] border border-outline-subtle bg-surface-card shadow-[var(--card-shadow)]">
    <div class="max-h-[min(65vh,45rem)] overflow-auto">
        <table
            class={[
                "w-full border-collapse text-ui-sm text-content-primary",
                "[font-variant-numeric:tabular-nums]",
                className,
            ]}
            {...rest}
        >
            {#if caption}
                <caption class="sr-only">{caption}</caption>
            {/if}
            <thead>{@render header()}</thead>
            <tbody>{@render body()}</tbody>
        </table>
    </div>
</div>
