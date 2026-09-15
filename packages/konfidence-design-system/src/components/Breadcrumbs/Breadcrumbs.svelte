<script lang="ts">
    /**
     * Breadcrumb trail — mirrors the Konfidence design-system `.crumbs`
     * primitive (components.css lines 533–537). The last item is always
     * marked as the current page and rendered as a non-interactive span
     * with `aria-current="page"`, regardless of whether it carries an
     * `href`. Callers may omit `href` on intermediate crumbs to render
     * them as plain text when a destination does not yet exist.
     */

    import type { BreadcrumbItem } from "./types.js";

    interface Props {
        /** Ordered items, root → current. Empty arrays render nothing. */
        items: BreadcrumbItem[];
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let { items, class: className }: Props = $props();
</script>

{#if items.length > 0}
    <nav
        class={["crumbs", className].filter(Boolean).join(" ")}
        aria-label="Breadcrumb"
        data-testid="breadcrumbs"
    >
        {#each items as item, index (index)}
            {@const isLast = index === items.length - 1}
            {#if index > 0}
                <span class="crumbs__sep" aria-hidden="true">/</span>
            {/if}
            {#if isLast || !item.href}
                <span
                    class={isLast ? "crumbs__current" : undefined}
                    aria-current={isLast ? "page" : undefined}
                >
                    {item.label}
                </span>
            {:else}
                <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Callers pass an already-resolved URL. -->
                <a href={item.href}>{item.label}</a>
            {/if}
        {/each}
    </nav>
{/if}

<style>
    .crumbs {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: var(--text-sm);
        color: var(--text-tertiary);
    }
    .crumbs a {
        color: var(--text-tertiary);
        text-decoration: none;
    }
    .crumbs a:hover {
        color: var(--text-link);
    }
    .crumbs__sep {
        color: var(--border-strong);
    }
    .crumbs__current {
        color: var(--text-primary);
        font-weight: var(--weight-medium);
    }
</style>
