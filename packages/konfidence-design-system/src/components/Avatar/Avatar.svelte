<script lang="ts">
    import type { Snippet } from "svelte";

    interface Props {
        /**
         * 1–2 character initials rendered when no `children` snippet is
         * provided. Ignored when `children` is set — that snippet then
         * owns the visible content.
         */
        initials?: string;
        /** Adds the amber orbit ring (`.avatar--orbit`) signature. */
        orbit?: boolean;
        /** Accessible label. Renders as `aria-label` on the element. */
        ariaLabel?: string;
        /**
         * Root element tag. Defaults to `"div"` (decorative use, e.g.
         * inside a `MenuHeader`); switch to `"button"` when the avatar
         * is interactive (e.g. wired to a Skeleton `Menu.Trigger`
         * `element` snippet) so native Enter/Space activation and the
         * default focusable/tabbable behaviour come along for free.
         */
        as?: "div" | "button";
        /**
         * Extra class names for layout tweaks. Widely typed so
         * framework-owned attribute bags (Zag `getTriggerProps()`,
         * Skeleton's `ClassValue`) spread through without friction.
         */
        class?: unknown;
        /**
         * Optional override for the visible content. Rendered instead of
         * `initials`. Used e.g. inside a `.menu__header` block where the
         * avatar is decorative (initials handled elsewhere).
         */
        children?: Snippet;
        /**
         * Any additional attributes are forwarded to the root element.
         * Kept loosely typed on purpose so that framework-owned attribute
         * bags (e.g. Zag's `getTriggerProps()`) can spread through
         * without fighting Svelte's `HTMLAttributes` union — the
         * assertion that the result renders as a valid element belongs
         * to the caller.
         */
        [key: string]: unknown;
    }

    let {
        initials,
        orbit = false,
        ariaLabel,
        as = "div",
        class: className,
        children,
        ...rest
    }: Props = $props();

    const composedClass = $derived(
        [
            "avatar flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-full border-0 bg-[image:var(--gradient-teal)] p-0 text-[length:var(--text-meta)] font-bold text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--border-focus)]",
            orbit
                ? "avatar--orbit relative before:absolute before:-inset-[3px] before:rounded-full before:border-[1.5px] before:border-[var(--orbit-ring)] before:opacity-40 before:content-['']"
                : "",
            typeof className === "string" && className.length > 0 ? className : "",
        ]
            .filter(Boolean)
            .join(" "),
    );
</script>

<svelte:element
    this={as}
    class={composedClass}
    aria-label={ariaLabel}
    {...as === "button" ? { type: "button" } : {}}
    {...rest}
>
    {#if children}
        {@render children()}
    {:else}
        {initials ?? ""}
    {/if}
</svelte:element>


