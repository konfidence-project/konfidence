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
        /** Bindable expansion, or controlled through onExpandedChange. */
        expanded?: boolean;
        onExpandedChange?: (expanded: boolean) => void;
        /** Single selection for descendant rows with a value. */
        selectedValue?: string;
        onValueChange?: (value: string) => void;
        /** Standard title; header can supply custom content instead. */
        label?: string;
        /** Noninteractive header content. */
        header?: Snippet;
        metadata?: Snippet;
        /** Optional always-visible content. Omit for a header-only collapsed group. */
        summary?: Snippet;
        /** Omit when there is no additional content to expand. */
        children?: Snippet;
        /** Optional focus-restoration target for the header. */
        triggerId?: string;
        class?: string;
    }

    let {
        ariaLabel,
        expanded = $bindable(false),
        onExpandedChange,
        selectedValue = $bindable<string | undefined>(undefined),
        onValueChange,
        label,
        header,
        metadata,
        summary,
        children,
        triggerId,
        class: className,
    }: Props = $props();
    const headerClass = "flex w-full min-w-0 items-start gap-2 bg-transparent px-4 py-3 text-left text-compact leading-[1.3] text-content-primary [overflow-wrap:anywhere]";

    setGroupSelection({
        select(value) {
            if (selectedValue === value) {
                return false;
            }
            selectedValue = value;
            onValueChange?.(value);
            return true;
        },
        get selectedValue() {
            return selectedValue;
        },
    });

    const changeExpanded = (open: boolean) => {
        if (expanded === open) {
            return;
        }
        if (onExpandedChange) {
            onExpandedChange(open);
        } else {
            expanded = open;
        }
    };
</script>

{#snippet headingContent()}
    <span class="min-w-0 grow">
        <span class="block font-semibold">{#if header}{@render header()}{:else}{label ?? ariaLabel}{/if}</span>
        {#if metadata}<span class="mt-1 flex flex-wrap items-center gap-2 font-normal leading-normal text-content-secondary">{@render metadata()}</span>{/if}
    </span>
{/snippet}

<Collapsible open={expanded && children !== undefined} onOpenChange={({ open }) => changeExpanded(open)}
    class={["flex flex-col items-stretch gap-0 overflow-hidden rounded-container border border-outline-subtle bg-surface-card text-compact leading-normal text-content-primary", className]}
    role="group" aria-label={ariaLabel}>
    {#if children}
        <Collapsible.Trigger id={triggerId} aria-label={ariaLabel}
            class={[headerClass, "cursor-pointer hover:bg-surface-subtle focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-outline-focus"]}>
            <ui5-icon aria-hidden="true" class="mt-0.5 size-4 shrink-0 text-content-tertiary" name={expanded ? "slim-arrow-down" : "slim-arrow-right"}></ui5-icon>
            {@render headingContent()}
        </Collapsible.Trigger>
    {:else}
        <div id={triggerId} class={headerClass} tabindex="-1">{@render headingContent()}</div>
    {/if}
    {#if summary}<div class="border-t border-outline-subtle">{@render summary()}</div>{/if}
    {#if children}
        <Collapsible.Content class="block w-full border-t border-outline-subtle [&>*+*]:border-t [&>*+*]:border-outline-subtle">
            {@render children()}
        </Collapsible.Content>
    {/if}
</Collapsible>
