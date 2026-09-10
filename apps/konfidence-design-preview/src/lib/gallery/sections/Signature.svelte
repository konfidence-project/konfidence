<script lang="ts">
    import { OrbitLoader, StatusBadge } from "@konfidence/design-system/components";
    import Sample from "../Sample.svelte";
    import Section from "../Section.svelte";
</script>

<Section
    id="signature"
    title="Signature"
    subtitle="Konfidence design language — glowing planet, orbit motion, amber-vs-teal semantics."
    status="implemented"
>
    <div class="grid">
        <Sample title="Orbit loader" description="Animated brand mark; live-region label for AT.">
            <OrbitLoader label="Deploying…" />
        </Sample>

        <Sample title="Amber = action" description="Reserved for primary calls-to-action.">
            <StatusBadge status="promoting">Deploy</StatusBadge>
            <StatusBadge status="deploying">Deploying</StatusBadge>
        </Sample>

        <Sample title="Teal = information" description="Neutral, informative accents.">
            <StatusBadge status="healthy">Healthy</StatusBadge>
            <StatusBadge status="queued">Queued</StatusBadge>
        </Sample>
    </div>

    <p class="rule">
        <b>Rule:</b> amber is only used to prompt an action (buttons, focus glow, active states);
        teal is used for information, background accents, and secondary highlights. Never mix them
        in the same affordance.
    </p>
</Section>

<style>
    .grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        gap: 16px;
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
