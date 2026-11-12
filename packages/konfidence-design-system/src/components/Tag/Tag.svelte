<script lang="ts">
    import type { Snippet } from "svelte";
    import type { HTMLAnchorAttributes, HTMLAttributes, HTMLButtonAttributes } from "svelte/elements";
    import type { TagColor, TagSize } from "./types.js";
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/AllIcons.js";

    interface CommonProps {
        color?: TagColor;
        /** S = 20px, M = 24px (default), independent of density. */
        size?: TagSize;
        /** Use the meta/monospace typeface for IDs and hashes. */
        mono?: boolean;
        /** Optional SAP icon name; omit for a text-only tag. */
        icon?: string;
        /** Allow long labels to wrap. Single-line by default. */
        wrap?: boolean;
        class?: string;
        children: Snippet;
    }
    type StaticProps = CommonProps & { interactive?: false; href?: never; disabled?: never } & Omit<HTMLAttributes<HTMLSpanElement>, "class" | "children" | "onclick">;
    type ButtonProps = CommonProps & { interactive: true; href?: never } & Omit<HTMLButtonAttributes, "class" | "children">;
    type LinkProps = CommonProps & { href: string; interactive?: true; disabled?: boolean } & Omit<HTMLAnchorAttributes, "class" | "children" | "href">;
    type Props = StaticProps | ButtonProps | LinkProps;
    let { color = "neutral", size = "md", mono = false, icon, wrap = false, interactive = false, href, disabled = false, class: className, children, ...rest }: Props = $props();
    const classes = $derived(["tag", `tag--${color}`, `tag--${size}`, mono && "tag--mono", wrap && "tag--wrap", (interactive || href !== undefined) && "tag--interactive", className]);
</script>
{#snippet content()}
    {#if icon}<ui5-icon class="tag__icon" name={icon} aria-hidden="true"></ui5-icon>{/if}
    <span class="tag__label">{@render children()}</span>
{/snippet}
{#if href !== undefined}
    <a {...rest as HTMLAnchorAttributes} class={classes} href={disabled ? undefined : href} aria-disabled={disabled ? "true" : undefined} onclick={disabled ? undefined : (rest as HTMLAnchorAttributes).onclick}>
        {@render content()}
    </a>
{:else if interactive}
    <button {...rest as HTMLButtonAttributes} class={classes} type={(rest as HTMLButtonAttributes).type ?? "button"} {disabled}>
        {@render content()}
    </button>
{:else}
    <span {...rest as HTMLAttributes<HTMLSpanElement>} class={classes}>
        {@render content()}
    </span>
{/if}
<style>
    .tag {
        --tag-fg: var(--text-secondary); --tag-bg: var(--surface-sunken); --tag-bd: var(--border-subtle);
        display: inline-flex; align-items: center; gap: 5px; box-sizing: border-box; max-width: 100%;
        min-height: var(--tag-height-md); padding: var(--tag-py-md) var(--tag-px-md);
        font-family: var(--font-sans); font-size: var(--font-size-meta); font-weight: var(--weight-medium);
        line-height: var(--tag-line-height); white-space: nowrap; text-decoration: none; vertical-align: middle;
        color: var(--tag-fg); background: var(--tag-bg); border: 1px solid var(--tag-bd); border-radius: var(--tag-radius);
    }
    .tag--sm { min-height: var(--tag-height-sm); padding: var(--tag-py-sm) var(--tag-px-sm); }
    .tag--mono { font-family: var(--font-mono); }
    .tag__icon, .tag :global(svg) { width: var(--icon-xs); height: var(--icon-xs); flex-shrink: 0; }
    .tag__icon { color: inherit; }
    .tag__label { min-width: 0; }
    .tag--wrap .tag__label { white-space: normal; overflow-wrap: anywhere; }
    .tag--interactive:is(button, a[href]) { cursor: pointer; touch-action: manipulation; transition: background var(--motion-fast) cubic-bezier(var(--ease)), border-color var(--motion-fast) cubic-bezier(var(--ease)), box-shadow var(--motion-fast) cubic-bezier(var(--ease)); }
    @media (hover: hover) {
        .tag--interactive:is(button, a[href]):not(:disabled):not([aria-disabled="true"]):hover { background: color-mix(in srgb, var(--tag-fg) 10%, var(--tag-bg)); border-color: color-mix(in srgb, var(--tag-fg) 55%, var(--tag-bd)); filter: none; }
    }
    .tag--interactive:is(button, a[href]):not(:disabled):not([aria-disabled="true"]):active { background: color-mix(in srgb, var(--tag-fg) 16%, var(--tag-bg)); }
    .tag--interactive:focus-visible { outline: none; box-shadow: var(--tag-focus, var(--focus-ring)); }
    .tag--teal.tag--interactive, a.tag--interactive { --tag-focus: var(--focus-ring-teal); }
    .tag--interactive:disabled, .tag--interactive[aria-disabled="true"] { opacity: var(--disabled-opacity); cursor: default; box-shadow: none; }
    @media (prefers-reduced-motion: reduce) { .tag { transition: none; } }
    .tag--red { --tag-fg: var(--tag-red-fg); --tag-bg: var(--tag-red-bg); --tag-bd: var(--tag-red-bd); }
    .tag--orange { --tag-fg: var(--tag-orange-fg); --tag-bg: var(--tag-orange-bg); --tag-bd: var(--tag-orange-bd); }
    .tag--amber { --tag-fg: var(--tag-amber-fg); --tag-bg: var(--tag-amber-bg); --tag-bd: var(--tag-amber-bd); }
    .tag--lime { --tag-fg: var(--tag-lime-fg); --tag-bg: var(--tag-lime-bg); --tag-bd: var(--tag-lime-bd); }
    .tag--green { --tag-fg: var(--tag-green-fg); --tag-bg: var(--tag-green-bg); --tag-bd: var(--tag-green-bd); }
    .tag--teal { --tag-fg: var(--tag-teal-fg); --tag-bg: var(--tag-teal-bg); --tag-bd: var(--tag-teal-bd); }
    .tag--cyan { --tag-fg: var(--tag-cyan-fg); --tag-bg: var(--tag-cyan-bg); --tag-bd: var(--tag-cyan-bd); }
    .tag--blue { --tag-fg: var(--tag-blue-fg); --tag-bg: var(--tag-blue-bg); --tag-bd: var(--tag-blue-bd); }
    .tag--indigo { --tag-fg: var(--tag-indigo-fg); --tag-bg: var(--tag-indigo-bg); --tag-bd: var(--tag-indigo-bd); }
    .tag--violet { --tag-fg: var(--tag-violet-fg); --tag-bg: var(--tag-violet-bg); --tag-bd: var(--tag-violet-bd); }
    .tag--pink { --tag-fg: var(--tag-pink-fg); --tag-bg: var(--tag-pink-bg); --tag-bd: var(--tag-pink-bd); }
    .tag--slate { --tag-fg: var(--tag-slate-fg); --tag-bg: var(--tag-slate-bg); --tag-bd: var(--tag-slate-bd); }
</style>
