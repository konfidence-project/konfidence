# `@konfidence/design-system`

Konfidence dashboard design system. Layered on top of Tailwind CSS v4 and
Skeleton v5, provides:

- **Design tokens** — colour ramps, spacing, typography, motion, radii,
  semantic tokens. Light-mode tokens live on `:root`; dark-mode tokens
  apply under `[data-mode="dark"]` on any element under
  `<html data-theme="konfidence">`, plus `[data-mode="system"]` inside
  a `@media (prefers-color-scheme: dark)` block (Skeleton pattern).
- **Skeleton theme** — colour ramps for `data-theme="konfidence"`.
- **Custom components** — Konfidence-specific CSS classes with no
  Skeleton equivalent (orbit, phases, diff, timelines, charts, status
  badges, tags, icon chips, …).
- **`.btn` styles** — Konfidence gradient fill, amber glow, hover
  lift; scoped inside `Button.svelte` (colocated with the component)
  so consumers get them via `<Button>`, not raw `.btn` markup.
- **Svelte components** — Tier-1 wrappers over the CSS layer:
  `Button`, `Brandbar`, `OrbitLoader`, `StatusBadge`.

The package is workspace-only (`"private": true`); it ships TypeScript
and Svelte source without a build step and relies on the consumer's
Vite + `@sveltejs/vite-plugin-svelte` to compile.

## Install

```jsonc
// apps/<your-app>/package.json
{
  "dependencies": {
    "@konfidence/design-system": "workspace:*",
  },
}
```

## Wire the stylesheet

Tailwind and Skeleton are peer dependencies. The consumer's `app.css`
imports them first, then layers the Konfidence styles on top:

```css
@import "tailwindcss";
@import "@skeletonlabs/skeleton";
@import "@skeletonlabs/skeleton-svelte";
@custom-variant dark {
  &:where([data-mode="dark"], [data-mode="dark"] *) {
    @slot;
  }
  @media (prefers-color-scheme: dark) {
    &:where([data-mode="system"], [data-mode="system"] *) {
      @slot;
    }
  }
}
@import "@konfidence/design-system/styles";
```

Fine-grained subpaths are available if the default order does not
fit (`@konfidence/design-system/styles/tokens`, `/styles/skeleton`,
`/styles/custom`).

## Wire the theme bootstrap

Theme resolution and persistence live in the consuming application
(the design system stays runtime-free) — see
`apps/konfidence-ui/src/lib/theme/` for the reference wiring:

- a synchronous inline `<script>` in `app.html` that resolves the
  theme before the first paint (so reloads never flash the wrong
  theme), and
- a reactive `ThemeStore` used at runtime to read/toggle/persist the
  theme.

The `data-theme="konfidence"` + `data-mode` selectors shipped here
are the contract those runtimes target.

## Components

| Component     | CSS classes it wraps                             | Purpose                                                                    |
| ------------- | ------------------------------------------------ | -------------------------------------------------------------------------- |
| `Button`      | `.btn`, `.btn--{primary,secondary,ghost,danger}` | Renders `<button>` or `<a>`; forwards `disabled`, `aria-*`, click handler. |
| `Brandbar`    | (Tailwind arbitrary-value utilities)             | The amber-teal aurora strip at the top of every screen.                    |
| `OrbitLoader` | (Tailwind arbitrary-value utilities)             | Live-region loading indicator with an accessible label.                    |
| `StatusBadge` | `.badge`, `.badge--<status>`                     | Passes `status` through to the class list; the API owns the vocabulary.    |

```svelte
<script lang="ts">
  import { Button, StatusBadge } from "@konfidence/design-system/components";
</script>

<Button variant="primary" onclick={deploy}>Deploy</Button>

<StatusBadge status="deploying">Deploying</StatusBadge>
```

## Linux screenshots

Each component lives in `src/components/<Component>/` with its tests,
harness, and screenshots. `pnpm ds:test` refreshes screenshot baselines
automatically; visual changes are reviewed as PNG diffs rather than test
failures. Only Linux PNGs are tracked by Git.

Screenshot coverage includes all four Button variants, the Brandbar,
OrbitLoader's default and custom labels, and StatusBadge's seven styled
statuses plus an unknown-status fallback, with and without the dot.
Every case runs in light and dark mode.

ButtonHarness and StatusBadgeHarness are test-only fixtures that supply
Svelte `children` snippets. Brandbar and OrbitLoader render directly in
their tests because they take ordinary props without snippets.

To regenerate the colocated Linux PNGs, run from the repository root:

```sh
pnpm ds:screenshots:generate
```

The generator builds `Dockerfile.screenshots` and runs `pnpm ds:test`
inside Linux. After the tests pass, it copies Linux PNGs directly into
each component's `__screenshots__/` directory and removes obsolete Linux
baselines. Review and commit those changes with the component or stylesheet
changes. Other platform screenshots remain untouched.

To verify freshness without modifying the checkout:

```sh
pnpm ds:screenshots:check
```

The scripts live in `hack/`. The check script asks the generator to export
into a temporary directory, compares filenames and contents with the
colocated baselines, and cleans up. The normal generate command needs no
temporary directory. Both commands remove their containers on exit.

Docker is required. Both commands use `linux/arm64`, matching the
`ubuntu-24.04-arm` CI runner and running natively on Apple Silicon.
The image contains the checkout, so remote Docker daemons work without
bind mounts. `Dockerfile.screenshots.dockerignore` excludes host dependencies
and existing screenshots. Playwright package versions are defined in the
root `pnpm-workspace.yaml` catalog and reused through `catalog:` dependencies.
Keep the image's Playwright version aligned with that catalog and the
lockfile. Native macOS `pnpm ds:test` runs refresh only ignored
macOS screenshots.

### PR freshness check

Frontend CI runs `pnpm ds:screenshots:check` on `ubuntu-24.04-arm`.
It fails for changed images, new images without committed baselines, and
obsolete baselines whose tests no longer generate them.

When this step fails, run `pnpm ds:screenshots:generate`, review the PNG
additions, changes, and deletions, and commit them. CI never commits changes
to your branch.

Make `Test Design System` a required status check in the repository's
branch rules to block merging when baselines are stale.

## Roadmap

The design system will grow in the following order:

1. **`tokens.json` + generator** — import
   `tokens/tokens.json` (W3C DTCG) and `build-tokens.mjs` from the
   external `konfidence-design` repo into
   `packages/konfidence-design-system/tokens/`; regenerate
   `src/styles/tokens.css` (and `konfidence.skeleton.css`) on `verify`
   with a diff-check that mirrors the `api:check` pattern.
2. **More components** — grown per feature, never speculatively.
   Cards (`Card`, `KPI`), tables (`Tag`), phases (`Phases`), timeline,
   diff.
3. **Preview app** — port `design-system.html` and `app-preview.html`
   from the external repo into `apps/konfidence-design-preview/` so
   the style guide never drifts from the package.

## Contributing

Interactive primitives (buttons, links styled as buttons, badges,
tags, chips, form fields, dialogs, menus, tabs, toasts, tooltips)
belong in `src/components/` as Svelte components so accessibility and
keyboard behaviour live in one place.

Decorative / layout patterns (`.orbit`, `.phase`, `.diff`, `.timeline`,
charts, `.hero`, `.command`, `.filterbar`) stay in
`src/styles/konfidence.custom.css` as CSS classes; promote them to
components only when a real component API emerges.

Skeleton-provided primitives (Dialog, Popover, Tooltip, Menu, Tabs,
Accordion, Segmented Control, Switch, Toast, Pagination, Progress,
Slider, Steps, Avatar, App Bar, Navigation) are consumed directly
from `@skeletonlabs/skeleton-svelte`; wrap them here only when we
want to constrain their API.
