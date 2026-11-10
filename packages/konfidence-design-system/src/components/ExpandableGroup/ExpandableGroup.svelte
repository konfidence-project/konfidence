<script lang="ts">
    import type { Snippet } from "svelte";
    import { Collapsible } from "@skeletonlabs/skeleton-svelte";
    import { setGroupSelection } from "./selection-context.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/slim-arrow-right.js";
    import "@ui5/webcomponents-icons/dist/slim-arrow-down.js";

    interface Props {
        /** Accessible name of the group and its expansion trigger. */
        ariaLabel: string;
        /** Bindable expansion; defaults to collapsed. */
        expanded?: boolean;
        /** Bindable single selection for descendant rows. */
        selectedValue?: string;
        /** Free header content; independent controls must stop click propagation. */
        header: Snippet;
        /** Omit when there is no additional content to expand. */
        children?: Snippet;
        /** Optional focus-restoration target for the header. */
        triggerId?: string;
    }

    let {
        ariaLabel,
        expanded = $bindable(false),
        selectedValue = $bindable<string | undefined>(undefined),
        header,
        children,
        triggerId,
    }: Props = $props();
    const headerClass = "flex w-full min-w-0 items-center bg-transparent px-4 py-3 text-left text-compact leading-[1.3] text-content-primary [overflow-wrap:anywhere]";

    setGroupSelection({
        select(value) {
            selectedValue = value;
        },
        get selectedValue() {
            return selectedValue;
        },
    });

</script>

<Collapsible open={expanded && children !== undefined} onOpenChange={({ open }) => expanded = open}
    class="flex flex-col items-stretch gap-0 overflow-hidden rounded-container border border-outline-subtle bg-surface-card text-compact leading-normal text-content-primary"
    role="group" aria-label={ariaLabel}>
    {#if children}
        <!-- The native chevron supplies keyboard disclosure; the layout adds a pointer target. -->
        <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
        <div class={[headerClass, "cursor-pointer hover:bg-surface-subtle"]} onclick={() => expanded = !expanded}>
            <Collapsible.Trigger id={triggerId} aria-label={ariaLabel} onclick={(event) => event.stopPropagation()}
                class="-my-1 -ml-2 flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-control focus-visible:outline-2 focus-visible:outline-outline-focus">
                <ui5-icon aria-hidden="true" class="size-4 text-content-tertiary" name={expanded ? "slim-arrow-down" : "slim-arrow-right"}></ui5-icon>
            </Collapsible.Trigger>
            <div class="min-w-0 grow">{@render header()}</div>
        </div>
    {:else}
        <div id={triggerId} class={headerClass} tabindex="-1"><div class="min-w-0 grow">{@render header()}</div></div>
    {/if}
    {#if children}
        <Collapsible.Content class="block w-full border-t border-outline-subtle [&>*+*]:border-t [&>*+*]:border-outline-subtle">
            {@render children()}
        </Collapsible.Content>
    {/if}
</Collapsible>
