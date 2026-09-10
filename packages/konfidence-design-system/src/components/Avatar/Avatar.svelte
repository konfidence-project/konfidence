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
         * Any additional attributes are forwarded to the root `<div>`. Kept
         * loosely typed on purpose so that framework-owned attribute bags
         * (e.g. Zag's `getTriggerProps()`) can spread through without
         * fighting Svelte's `HTMLAttributes` union — the assertion that the
         * result renders as a valid element belongs to the caller.
         */
        [key: string]: unknown;
    }

    let {
        initials,
        orbit = false,
        ariaLabel,
        class: className,
        children,
        ...rest
    }: Props = $props();

    const composedClass = $derived([
        "avatar",
        orbit ? "avatar--orbit" : "",
        typeof className === "string" && className.length > 0 ? className : "",
    ].filter(Boolean).join(" "));
</script>

<div class={composedClass} aria-label={ariaLabel} {...rest}>
    {#if children}
        {@render children()}
    {:else}
        {initials ?? ""}
    {/if}
</div>

<style>
    .avatar {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        background: var(--gradient-teal);
        color: #fff;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: var(--text-meta);
        font-weight: var(--weight-bold);
        cursor: pointer;
        flex-shrink: 0;
    }
    .avatar:focus-visible {
        outline: 2px solid var(--border-focus);
        outline-offset: 2px;
    }

    /* orbit ring signature */
    .avatar--orbit {
        position: relative;
    }
    .avatar--orbit::before {
        content: "";
        position: absolute;
        inset: -3px;
        border-radius: 50%;
        border: 1.5px solid var(--orbit-ring);
        opacity: 0.4;
    }
</style>


