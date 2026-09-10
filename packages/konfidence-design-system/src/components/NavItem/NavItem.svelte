<script lang="ts">
    import type { HTMLAnchorAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";
    import "@ui5/webcomponents/dist/Icon.js";
    import "@ui5/webcomponents-icons/dist/AllIcons.js";

    interface Props extends Omit<HTMLAnchorAttributes, "class" | "href"> {
        /** Resolved destination URL. */
        href: string;
        /**
         * Whether this destination represents the current page. Sets
         * `.nav-item--active` and `aria-current="page"`.
         */
        active?: boolean;
        /** Leading SAP-icons name, e.g. `grid`. */
        icon?: string;
        /**
         * Optional numeric counter (`.nav-item__badge`). Values above 99
         * render as `99+`.
         */
        badge?: number;
        /** Extra class names for layout tweaks. */
        class?: string;
        /** Label snippet. */
        children: Snippet;
    }

    let {
        href,
        active = false,
        icon,
        badge,
        class: className,
        children,
        ...rest
    }: Props = $props();

    const composedClass = $derived([
        "nav-item",
        active ? "nav-item--active" : "",
        className ?? "",
    ].filter(Boolean).join(" "));

    const BADGE_MAX = 99;
    const badgeLabel = $derived.by((): string | undefined => {
        if (badge === undefined) {
            return undefined;
        }
        return badge > BADGE_MAX ? `${BADGE_MAX}+` : String(badge);
    });
</script>

<a
    class={composedClass}
    {href}
    aria-current={active ? "page" : undefined}
    {...rest}
>
    {#if icon}
        <ui5-icon name={icon}></ui5-icon>
    {/if}
    <span>{@render children()}</span>
    {#if badgeLabel !== undefined}
        <span class="nav-item__badge">{badgeLabel}</span>
    {/if}
</a>

<style>
    ui5-icon {
        width: 16px;
        height: 16px;
        color: currentColor;
    }

    .nav-item {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 8px var(--space-3);
        border-radius: var(--radius-md);
        color: var(--text-secondary);
        font-size: var(--text-sm);
        font-weight: var(--weight-medium);
        cursor: pointer;
        text-decoration: none;
        margin-bottom: 1px;
        position: relative;
    }
    .nav-item:hover {
        background: var(--surface-sunken);
        color: var(--text-primary);
    }
    .nav-item:focus-visible {
        outline: 2px solid var(--border-focus);
        outline-offset: -2px;
    }

    .nav-item--active {
        background: var(--amber-50);
        color: var(--amber-800);
        font-weight: var(--weight-semibold);
    }
    :global([data-mode="dark"]) .nav-item--active {
        background: rgba(255, 181, 48, 0.12);
        color: var(--amber-300);
    }
    @media (prefers-color-scheme: dark) {
        :global([data-mode="system"]) .nav-item--active {
            background: rgba(255, 181, 48, 0.12);
            color: var(--amber-300);
        }
    }
    .nav-item--active::before {
        content: "";
        position: absolute;
        left: 0;
        top: 6px;
        bottom: 6px;
        width: 3px;
        border-radius: var(--radius-pill);
        background: var(--gradient-amber);
    }

    .nav-item__badge {
        margin-left: auto;
        min-width: 18px;
        height: 18px;
        padding: 0 5px;
        border-radius: var(--radius-pill);
        background: var(--status-error-bg);
        color: var(--status-error-fg);
        font-size: 10px;
        font-weight: var(--weight-bold);
        display: flex;
        align-items: center;
        justify-content: center;
    }
</style>
