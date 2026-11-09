<script lang="ts">
    import type { Snippet } from "svelte";
    import { Collapsible } from "@skeletonlabs/skeleton-svelte";
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/slim-arrow-right.js";
    import "@ui5/webcomponents-icons/dist/slim-arrow-down.js";

    interface Props {
        /** Accessible name of the group and its expansion trigger. */
        ariaLabel: string;
        /** Controlled expansion; the consumer applies onExpandedChange. */
        expanded: boolean;
        onExpandedChange: (expanded: boolean) => void;
        /** Noninteractive header content. */
        header: Snippet;
        /** Optional always-visible content. Omit for a header-only collapsed group. */
        summary?: Snippet;
        /** Omit when there is no additional content to expand. */
        children?: Snippet;
        /** Optional focus-restoration target for the header. */
        triggerId?: string;
    }

    let { ariaLabel, expanded, onExpandedChange, header, summary, children, triggerId }: Props = $props();
    const headerClass = "flex w-full min-w-0 items-center gap-4 bg-transparent px-5 py-4 text-left text-content-primary [overflow-wrap:anywhere]";
</script>

<Collapsible open={expanded && children !== undefined} onOpenChange={({ open }) => onExpandedChange(open)}
    class={["flex flex-col items-stretch gap-0 overflow-hidden rounded-[var(--card-radius)] border border-outline-subtle shadow-elevation-xs", expanded ? "bg-surface-card" : "bg-surface-subtle"]}
    role="group" aria-label={ariaLabel}>
    {#if children}
        <Collapsible.Trigger id={triggerId} aria-label={ariaLabel}
            class={[headerClass, "cursor-pointer hover:bg-surface-subtle focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-outline-focus"]}>
            <ui5-icon aria-hidden="true" class="size-3.5 shrink-0 text-content-secondary" name={expanded ? "slim-arrow-down" : "slim-arrow-right"}></ui5-icon>
            <span class="min-w-0 grow">{@render header()}</span>
        </Collapsible.Trigger>
    {:else}
        <div id={triggerId} class={headerClass} tabindex="-1">{@render header()}</div>
    {/if}
    {#if summary}<div class="border-t border-outline-subtle">{@render summary()}</div>{/if}
    {#if children}
        <Collapsible.Content class="block w-full border-t border-outline-subtle [&>*+*]:border-t [&>*+*]:border-outline-subtle">
            {@render children()}
        </Collapsible.Content>
    {/if}
</Collapsible>
