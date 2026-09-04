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
