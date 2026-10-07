<script lang="ts">
    /**
     * Page header primitive — matches the Konfidence design-system
     * `.page-head` pattern (see design-system/app-preview.html) used by
     * every inner app screen. Deliberately spacing-only: no border,
     * no background band. Callers compose the header from three
     * optional pieces plus the required title:
     *
     *   • eyebrow  — snippet rendered ABOVE the title (breadcrumbs,
     *                back link, category label…)
     *   • title    — the H1 text; typography is locked to the DS
     *                display recipe (--font-size-h1 + --weight-display +
     *                --tracking-h1)
     *   • description — optional paragraph directly under the title,
     *                   in --font-size-compact / --text-secondary
     *   • actions  — snippet rendered to the right of the title row
     *                (primary button, filter buttons…)
     *
     * The `id` prop is forwarded onto the `<h1>` so callers can wire
     * `aria-labelledby` on their outer landmark.
     */

    import type { Snippet } from "svelte";

    interface Props {
        /** H1 text. Required — pages without a visible title violate the auditability rule. */
        title: string;
        /** Optional description shown directly under the title. */
        description?: string;
        /** Optional slot rendered above the title (e.g. breadcrumbs). */
        eyebrow?: Snippet;
        /** Optional right-aligned actions slot (e.g. primary button). */
        actions?: Snippet;
        /** id forwarded to the `<h1>` for aria-labelledby wire-up. */
        id?: string;
        /** Extra class names for layout tweaks. */
        class?: string;
    }

    let {
        title,
        description,
        eyebrow,
        actions,
        id,
        class: className,
    }: Props = $props();
</script>

<div class={className}>
    {#if eyebrow}
        <div class="mb-4">{@render eyebrow()}</div>
    {/if}
    <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
            <h1
                {id}
                class="m-0 text-h1 font-display tracking-[var(--tracking-h1)] leading-tight text-content-primary [overflow-wrap:anywhere]"
                data-testid="page-heading"
            >
                {title}
            </h1>
            {#if description}
                <p
                    class="m-0 mt-0.5 text-compact text-content-secondary"
                >
                    {description}
                </p>
            {/if}
        </div>
        {#if actions}
            <div class="flex flex-shrink-0 flex-wrap gap-3">
                {@render actions()}
            </div>
        {/if}
    </div>
</div>
