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
        "nav-item relative mb-px flex items-center gap-2.5 rounded-[var(--radius-md)] py-2 px-3 text-[length:var(--text-sm)] font-medium text-[var(--text-secondary)] no-underline cursor-pointer hover:bg-[var(--surface-sunken)] hover:text-[var(--text-primary)] focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--border-focus)]",
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
        <ui5-icon class="size-4 text-current" name={icon}></ui5-icon>
    {/if}
    <span>{@render children()}</span>
    {#if badgeLabel !== undefined}
        <span
            class="nav-item__badge ml-auto flex h-[18px] min-w-[18px] items-center justify-center rounded-[var(--radius-pill)] bg-[var(--status-error-bg)] px-[5px] text-[10px] font-bold text-[var(--status-error-fg)]"
        >{badgeLabel}</span>
    {/if}
</a>

<style>
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
</style>
