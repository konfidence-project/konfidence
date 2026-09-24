<script lang="ts">
    import type { Snippet } from "svelte";
    import type { HTMLAnchorAttributes, HTMLButtonAttributes } from "svelte/elements";

    interface CommonProps {
        /** Optional extra class names appended after the base class. */
        class?: string;
        /** Link label. */
        children?: Snippet;
    }

    type ButtonProps = CommonProps & {
        href?: undefined;
    } & Omit<HTMLButtonAttributes, "class" | "children">;

    type AnchorProps = CommonProps & {
        href: string;
    } & Omit<HTMLAnchorAttributes, "class" | "children" | "href">;

    type Props = ButtonProps | AnchorProps;

    let { class: className, children, ...rest }: Props = $props();

    const composedClass = $derived(className ? `link ${className}` : "link");
</script>

{#if "href" in rest && rest.href !== undefined}
    <a class={composedClass} {...rest as HTMLAnchorAttributes}>
        {@render children?.()}
    </a>
{:else}
    <button class={composedClass} type={(rest as HTMLButtonAttributes).type ?? "button"} {...rest as HTMLButtonAttributes}>
        {@render children?.()}
    </button>
{/if}

<style>
    .link {
        display: inline;
        margin: 0;
        padding: 0;
        border: 0;
        background: transparent;
        color: var(--text-link);
        font: inherit;
        line-height: inherit;
        text-decoration: underline;
        text-underline-offset: 3px;
        cursor: pointer;
    }

    .link:focus-visible {
        border-radius: var(--radius-sm);
        box-shadow: var(--focus-ring);
        outline: none;
    }

    .link:disabled {
        color: var(--text-disabled);
        cursor: not-allowed;
    }
</style>