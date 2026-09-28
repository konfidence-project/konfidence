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
    <div class="grid grid-cols-[repeat(auto-fit,minmax(260px,1fr))] gap-4">
        <Sample
            title="Amber = action"
            description="Reserved for calls-to-action, primary focus, active glow. Rendered here by the primary Button (--gradient-amber)."
        >
            <div class="signature-row flex flex-wrap items-center gap-3">
                <Button variant="primary">Deploy</Button>
                <span
                    class="signature-swatch inline-block size-10 shrink-0 rounded-full border border-outline-subtle"
                    style="background: var(--gradient-amber)"
                    aria-hidden="true"
                ></span>
                <code class="signature-token rounded bg-surface-subtle px-1.5 py-0.5 font-ui-mono text-meta text-content-secondary">--gradient-amber</code>
            </div>
        </Sample>

        <Sample
            title="Teal = information"
            description="Reserved for links, data points, running-state accents. Rendered here by a text link using --text-link."
        >
            <div class="signature-row flex flex-wrap items-center gap-3">
                <a class="signature-link text-ui-sm font-ui-semibold text-content-link no-underline hover:underline focus-visible:underline" href="#signature">View details →</a>
                <code class="signature-token rounded bg-surface-subtle px-1.5 py-0.5 font-ui-mono text-meta text-content-secondary">--text-link</code>
            </div>
        </Sample>
    </div>

    <p class="rule mt-4 border-l-[3px] border-[var(--accent-primary,var(--amber-500))] bg-surface-subtle px-3.5 py-3 text-ui-sm text-content-secondary">
        <b>Rule:</b> amber is only used to prompt an action (primary buttons, focus glow, active
        states); teal is used for information, links, data points, and running-state accents.
        Never mix them in the same affordance — a single primary button per context.
    </p>
</Section>
