# `@konfidence/design-system`

Konfidence dashboard design system. Layered on top of Tailwind CSS v4 and
Skeleton v5, provides:

- **Design tokens** — colour ramps, spacing, typography, motion, radii,
  semantic tokens. Light tokens live on `:root`; dark tokens apply on
  `<html data-theme="konfidence" class="dark">`.
- **Skeleton theme** — colour ramps for `data-theme="konfidence"`.
- **Component styling** — semantic Tailwind utilities for ordinary layout,
  typography and color; small scoped `<style>` blocks remain for complex
  treatments and nested component overrides.
- **Svelte components** — Tier-1 wrappers over the CSS layer:
  `Button`, `Brandbar`, `OrbitLoader`, `StatusBadge`.
- **Icons** — [SAP-icons](https://sap.github.io/ui5-webcomponents/nightly/v2/components/Icon/)
  via `<ui5-icon name="…">`. Components that render icons import
  `@ui5/webcomponents/dist/Icon.js` and
  `@ui5/webcomponents-icons/dist/AllIcons.js` themselves, so consumers
  only pass a SAP-icons name (e.g. `org-chart`, `slim-arrow-down`).

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
@custom-variant dark (&:where(.dark, .dark *));
@import "@konfidence/design-system/styles";
```

Fine-grained subpaths are available if the default order does not
fit (`@konfidence/design-system/styles/tokens`, `/styles/skeleton`).

### Style ownership

Keep the four stylesheets focused: `tokens.css` exports raw palette, semantic
light/dark, status, typography, gradient and shadow values (also available
without Tailwind via `/styles/tokens`). `konfidence.skeleton.css` configures
Skeleton's ramps, contrast, root colors, base type, radii and spacing using
those sources; its light/dark root values use explicit mode sources, not
mode-switching semantic aliases. `konfidence.theme.css` maps additional
semantic Tailwind utilities and intentional standard weight overrides to
the tokens. `index.css` imports them and owns only shared layout/global rules;
Skeleton's globals set the root background and body typography/color; the
body retains its own canvas background for app-shell coverage. Keep one-off styling
in the component instead of adding unused theme aliases.

### Semantic utilities

The main stylesheet imports `konfidence.theme.css` once. Its Tailwind v4
`@theme inline` mappings resolve existing CSS tokens at the element, so
`text-content-primary`, `bg-surface-card`, `border-outline-subtle`,
`text-status-error-fg` and `shadow-elevation-xs` respond to the root `.dark`
class. Use `text-meta` (12px), `text-compact` (13px), or `text-body` (14px);
Skeleton's `text-sm` remains 14px with its own line-height. Our named sizes
set only font-size, leaving line-height inherited unless specified. Their
raw plain-CSS sources are `--font-size-meta`, `--font-size-compact`, and
`--font-size-body` (likewise for headings, display and hero). Use Skeleton's
`rounded-base` (10px) and `rounded-container` (14px), `rounded-full` for
pills, `font-medium`/`font-semibold`/`font-bold` for shared weights,
`font-display` for the display weight and `font-mono` for the monospace stack.
One-off values and dynamic swatches still use CSS vars.

## Wire the theme bootstrap

The consuming app owns mode resolution and persistence. With mode-watcher,
mount `<ModeWatcher defaultTheme="konfidence" />` in the root layout. It
manages `data-theme` and toggles the root `.dark` class for resolved
appearance, including when the user chooses system mode.
`ColorModeSelect` accepts the selected `value`, an `onValueChange` callback,
and an optional resolved `resolvedMode` for its trigger icon. It does not
manage mode or storage itself.

## Components

| Component                                            | CSS classes it wraps                             | Purpose                                                                                                                          |
| ---------------------------------------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| `Button`                                             | `.btn`, `.btn--{primary,secondary,ghost,danger}` | Renders `<button>` or `<a>`; forwards `disabled`, `aria-*`, click handler.                                                       |
| `Brandbar`                                           | Tailwind utilities                               | The amber-teal aurora strip at the top of every screen.                                                                          |
| `EmptyState`                                         | Tailwind utilities                               | Neutral shell for empty, info, and error placeholders (icon + text + action).                                                    |
| `OrbitLoader`                                        | Tailwind utilities                               | Live-region loading indicator with an accessible label.                                                                          |
| `SearchInput`                                        | Tailwind utilities                               | Icon-prefixed search field with inline clear button.                                                                             |
| `SidePanel`                                          | Tailwind utilities                               | Right-anchored detail drawer built on native `<dialog>` (focus trap + `Esc` close).                                              |
| `StatusBadge`                                        | `.badge`, `.badge--<status>`                     | Passes `status` through to the class list; the API owns the vocabulary.                                                          |
| `Table` + `TableRow`, `TableCell`, `TableHeaderCell` | Tailwind utilities                               | Scrollable, sticky-header data-table primitives; `TableRow` accepts an `onselect` callback for click / Enter / Space activation. |

```svelte
<script lang="ts">
  import { Button, StatusBadge } from "@konfidence/design-system/components";
</script>

<Button variant="primary" onclick={deploy}>Deploy</Button>

<StatusBadge status="deploying">Deploying</StatusBadge>
```

## Linux screenshots

Each component lives in `src/components/<Component>/` with its tests,
test container, and screenshots. `pnpm ds:test` refreshes screenshot baselines
automatically; visual changes are reviewed as PNG diffs rather than test
failures. Only Linux PNGs are tracked by Git.

Screenshot coverage includes all four Button variants, the Brandbar,
OrbitLoader's default and custom labels, and StatusBadge's seven styled
statuses plus an unknown-status fallback, with and without the dot.
Every case runs in light and dark mode.

ButtonTestContainer and StatusBadgeTestContainer are test-only fixtures
(colocated in each component's `__tests__/` directory) that supply Svelte
`children` snippets. Brandbar and OrbitLoader render directly in their tests
because they take ordinary props without snippets.

To regenerate the colocated Linux PNGs, run from the repository root:

```sh
pnpm ds:screenshots:generate
```

The shared generator builds `hack/Dockerfile.screenshots` and runs `screenshots:regenerate` in this workspace
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
bind mounts. `hack/Dockerfile.screenshots.dockerignore` excludes host dependencies
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

Every component's CSS lives inside its own `.svelte` file as a scoped
`<style>` block — colocated with the markup it styles, not in a shared
stylesheet. Build new decorative / layout patterns (phases, diff,
timeline, charts, hero, command palette, …) as a component from the
start; there is no separate CSS-only staging file to drop rules into
before a component exists.

Skeleton-provided primitives (Dialog, Popover, Tooltip, Menu, Tabs,
Accordion, Segmented Control, Switch, Toast, Pagination, Progress,
Slider, Steps, Avatar, App Bar, Navigation) are consumed directly
from `@skeletonlabs/skeleton-svelte`; wrap them here only when we
want to constrain their API.
