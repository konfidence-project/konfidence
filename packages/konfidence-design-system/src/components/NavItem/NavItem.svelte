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

    const BASE_CLASS = "nav-item relative mb-px flex items-center gap-2.5 rounded-control py-2 px-3 text-ui-sm no-underline cursor-pointer not-[.nav-item--active]:hover:bg-surface-sunken not-[.nav-item--active]:hover:text-content-primary focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-outline-focus";

    const BADGE_MAX = 99;
    const badgeLabel = $derived.by((): string | undefined => {
        if (badge === undefined) {
            return undefined;
        }
        return badge > BADGE_MAX ? `${BADGE_MAX}+` : String(badge);
    });
</script>

<a
    class={[
        BASE_CLASS,
        active ? "nav-item--active bg-selection-bg text-selection-fg font-ui-semibold before:absolute before:top-1.5 before:bottom-1.5 before:left-0 before:w-[3px] before:rounded-pill-ui before:bg-[image:var(--gradient-amber)] before:content-['']" : "font-medium text-content-secondary",
        className,
    ]}
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
            class="nav-item__badge ml-auto flex h-[18px] min-w-[18px] items-center justify-center rounded-pill-ui bg-status-error-bg px-[5px] text-[10px] font-bold text-status-error-fg"
        >{badgeLabel}</span>
    {/if}
</a>
