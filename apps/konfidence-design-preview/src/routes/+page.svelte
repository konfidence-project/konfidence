<script lang="ts">
    import { Button, OrbitLoader } from "@konfidence/design-system/components";
    import { resolve } from "$app/paths";
</script>

<div class="hero">
    <div class="hero__glow" aria-hidden="true">
        <OrbitLoader label="" />
    </div>
    <h1 class="hero__title">
        Design with <span class="hero__accent">confidence.</span>
    </h1>
    <p class="hero__lead">
        A locally-startable, interactive explorer for the Konfidence design system. Browse every
        design token, component and pattern shipped by <code>@konfidence/design-system</code> —
        live-rendered from the same source the dashboard consumes.
    </p>
    <div class="hero__ctas">
        <Button variant="primary" href={resolve("/design-system")}>Explore the style guide</Button>
        <Button
            variant="ghost"
            href="https://github.com/konfidence-project/konfidence"
            target="_blank"
            rel="noopener"
        >
            Source
        </Button>
    </div>
</div>

<div class="cards">
    <a class="card" href={resolve("/design-system")}>
        <span class="card__eyebrow">Style guide</span>
        <span class="card__title">All components</span>
        <span class="card__desc">
            Every shipped component in <code>@konfidence/design-system</code> — buttons, icons, app
            shell, status badges, menus, orbit loader — alongside the foundations they build on
            (colors, gradients, typography, spacing, radii).
        </span>
    </a>
    <div class="card card--muted">
        <span class="card__eyebrow">App preview</span>
        <span class="card__title">Coming later</span>
        <span class="card__desc">
            A full dashboard demo (like <code>app-preview.html</code> in the reference design) will
            live here once more data-block primitives ship.
        </span>
    </div>
    <a
        class="card"
        href="https://github.com/konfidence-project/konfidence/tree/main/packages/konfidence-design-system"
        target="_blank"
        rel="noopener"
    >
        <span class="card__eyebrow">Docs</span>
        <span class="card__title">Package README</span>
        <span class="card__desc">
            Install, wire the stylesheet, contribute new components. The design-system package's
            README documents the roadmap and contribution rules.
        </span>
    </a>
</div>

<style>
    .hero {
        position: relative;
        padding: 64px 32px 48px;
        text-align: center;
        border-radius: 20px;
        background: var(--gradient-hero-bg, var(--surface-subtle));
        border: 1px solid var(--border-subtle);
        overflow: hidden;
    }

    .hero__glow {
        display: flex;
        justify-content: center;
        margin-bottom: 16px;
    }

    .hero__title {
        margin: 0 0 12px;
        font-size: var(--text-hero, 48px);
        font-weight: var(--weight-display, 600);
        letter-spacing: -1.5px;
        color: var(--text-primary);
    }

    .hero__accent {
        background: var(--gradient-planet, linear-gradient(180deg, #ffcb49, #ff960c));
        -webkit-background-clip: text;
        background-clip: text;
        color: transparent;
    }

    .hero__lead {
        margin: 0 auto 24px;
        max-width: 620px;
        color: var(--text-secondary);
        font-size: var(--text-body);
        line-height: 1.6;
    }

    .hero__lead code,
    .card__desc code {
        font-family: var(--font-mono);
        font-size: 0.9em;
        padding: 1px 5px;
        border-radius: 4px;
        background: var(--surface-subtle);
        color: var(--text-primary);
    }

    .hero__ctas {
        display: inline-flex;
        gap: 12px;
    }

    .cards {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
        gap: 16px;
        margin-top: 32px;
    }

    .card {
        display: flex;
        flex-direction: column;
        gap: 8px;
        padding: 20px;
        border: 1px solid var(--border-subtle);
        border-radius: 14px;
        background: var(--surface-default);
        color: var(--text-primary);
        text-decoration: none;
        transition:
            transform var(--motion-fast, 120ms) var(--ease, ease),
            box-shadow var(--motion-base, 200ms) var(--ease, ease),
            border-color var(--motion-fast, 120ms) var(--ease, ease);
    }

    .card:hover:not(.card--muted) {
        transform: translateY(-1px);
        box-shadow: var(--shadow-md, 0 6px 22px rgba(0, 0, 0, 0.06));
        border-color: var(--accent-primary, var(--amber-500));
    }

    .card--muted {
        opacity: 0.65;
    }

    .card__eyebrow {
        font-size: var(--text-meta);
        font-weight: var(--weight-semibold, 600);
        text-transform: uppercase;
        letter-spacing: 0.06em;
        color: var(--text-tertiary, var(--text-secondary));
    }

    .card__title {
        font-size: var(--text-h3, 18px);
        font-weight: var(--weight-display, 600);
    }

    .card__desc {
        color: var(--text-secondary);
        font-size: var(--text-sm);
        line-height: 1.55;
    }
</style>
