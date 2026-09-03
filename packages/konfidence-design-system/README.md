# `@konfidence/design-system`

Konfidence dashboard design system. Layered on top of Tailwind CSS v4 and
Skeleton v5, provides:

- **Design tokens** — colour ramps, spacing, typography, motion, radii,
  semantic tokens for the `konfidence` (light) and `konfidence-dark`
  themes.
- **Skeleton theme** — colour ramps for `data-theme="konfidence"` and
  `data-theme="konfidence-dark"`.
- **Custom components** — Konfidence-specific CSS classes with no
  Skeleton equivalent (orbit, phases, diff, timelines, charts, status
  badges, tags, icon chips, …).
- **`.btn` overrides** — Konfidence gradient fill, amber glow, hover
  lift; imported after Skeleton so it wins at equal specificity.
- **Theme store** — a small runtime that resolves the initial theme
  from `?theme=`, `localStorage`, or a default, exposes it reactively,
  and persists user changes.
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
@custom-variant dark (&:where([data-theme="konfidence-dark"], [data-theme="konfidence-dark"] *));
@import "@konfidence/design-system/styles";
```

Fine-grained subpaths are available if the default order does not
fit (`@konfidence/design-system/styles/tokens`, `/styles/skeleton`,
`/styles/custom`, `/styles/buttons`).

## Wire the theme bootstrap

Two pieces cooperate: a synchronous inline `<script>` in `app.html`
that resolves the theme before the first paint (so reloads never
flash the wrong theme), and the reactive `ThemeStore` used at runtime.

```html
<!-- app.html -->
<html lang="en" data-theme="konfidence">
  <head>
    <!-- Paste the string emitted by buildInlineBootstrapScript(). A
         snapshot test in @konfidence/design-system asserts the two
         stay in sync. -->
    <script>
      /* injected verbatim */
    </script>
  </head>
  …
</html>
```

```ts
// anywhere in your app
import { themeStore, type Theme } from "@konfidence/design-system/theme";

themeStore.current; // 'konfidence' | 'konfidence-dark'
themeStore.set("konfidence-dark");
themeStore.toggle();
```

### Theme precedence (highest wins)

1. `?theme=konfidence|konfidence-dark` in the URL — applied, persisted
   to `localStorage`, and stripped from the URL via
   `history.replaceState` so refreshes are idempotent.
2. `localStorage["konfidence.theme"]` — verbatim.
3. `konfidence` (light) — the default.

## Components

| Component     | CSS classes it wraps                                                            | Purpose                                                                    |
| ------------- | ------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `Button`      | `.btn`, `.btn--{primary,secondary,ghost,danger}`                                | Renders `<button>` or `<a>`; forwards `disabled`, `aria-*`, click handler. |
| `Brandbar`    | (Tailwind arbitrary-value utilities)                                            | The amber-teal aurora strip at the top of every screen.                    |
| `OrbitLoader` | (Tailwind arbitrary-value utilities)                                            | Live-region loading indicator with an accessible label.                    |
| `StatusBadge` | `.badge`, `.badge--{healthy,warning,degraded,error,promoting,deploying,queued}` | Named domain state; requires visible text.                                 |

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
