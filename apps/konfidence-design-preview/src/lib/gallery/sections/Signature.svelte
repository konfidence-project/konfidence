<script lang="ts">
    import { Button } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="signature"
    title="Signature"
    subtitle="Konfidence design language — amber-vs-teal semantics, strict colour discipline."
    status="implemented"
>
    <div class="grid">
        <Sample
            title="Amber = action"
            description="Reserved for calls-to-action, primary focus, active glow. Rendered here by the primary Button (--gradient-amber)."
        >
            <div class="signature-row">
                <Button variant="primary">Deploy</Button>
                <span
                    class="signature-swatch"
                    style="background: var(--gradient-amber)"
                    aria-hidden="true"
                ></span>
                <code class="signature-token">--gradient-amber</code>
            </div>
        </Sample>

        <Sample
            title="Teal = information"
            description="Reserved for links, data points, running-state accents. Rendered here by a text link using --text-link."
        >
            <div class="signature-row">
                <a class="signature-link" href="#signature">View details →</a>
                <code class="signature-token">--text-link</code>
            </div>
        </Sample>
    </div>

    <p class="rule">
        <b>Rule:</b> amber is only used to prompt an action (primary buttons, focus glow, active
        states); teal is used for information, links, data points, and running-state accents.
        Never mix them in the same affordance — a single primary button per context.
    </p>
</Section>

<style>
    .grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
        gap: 16px;
    }

    .signature-row {
        display: flex;
        align-items: center;
        gap: 12px;
        flex-wrap: wrap;
    }

    .signature-swatch {
        display: inline-block;
        width: 40px;
        height: 40px;
        border-radius: 999px;
        border: 1px solid var(--border-subtle);
        flex-shrink: 0;
    }

    .signature-link {
        color: var(--text-link);
        font-weight: var(--weight-semibold, 600);
        text-decoration: none;
        font-size: var(--text-sm);
    }

    .signature-link:hover,
    .signature-link:focus-visible {
        text-decoration: underline;
    }

    .signature-token {
        font-family: var(--font-mono);
        font-size: var(--text-meta);
        padding: 2px 6px;
        border-radius: 4px;
        background: var(--surface-subtle);
        color: var(--text-secondary);
    }

    .rule {
        margin: 16px 0 0;
        padding: 12px 14px;
        border-left: 3px solid var(--accent-primary, var(--amber-500));
        background: var(--surface-subtle);
        color: var(--text-secondary);
        font-size: var(--text-sm);
    }
</style>
