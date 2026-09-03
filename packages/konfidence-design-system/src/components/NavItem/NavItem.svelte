<script lang="ts">
    import type { HTMLAnchorAttributes } from "svelte/elements";
    import type { Snippet } from "svelte";
    import Icon from "../Icon/Icon.svelte";
    import type { IconName } from "../Icon/icons.js";

    interface Props extends Omit<HTMLAnchorAttributes, "class" | "href"> {
        /** Resolved destination URL. */
        href: string;
        /**
         * Whether this destination represents the current page. Sets
         * `.nav-item--active` and `aria-current="page"`.
         */
        active?: boolean;
        /** Leading icon from the DS icon set. */
        icon?: IconName;
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
        <Icon name={icon} size={16} />
    {/if}
    <span>{@render children()}</span>
    {#if badgeLabel !== undefined}
        <span class="nav-item__badge">{badgeLabel}</span>
    {/if}
</a>
