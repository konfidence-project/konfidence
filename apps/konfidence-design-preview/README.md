# `konfidence-design-preview`

Interactive explorer for the Konfidence design system. The catalog opens at `/`
with a flat, alphabetically sorted sidebar and an implemented-only gallery by default.
Links such as `/#component-Table` scroll to their component without unmounting other
examples. Native links support new tabs, reloads and browser history. Older query
links (`/?component=Table`) and section bookmarks (`/#lists`) remain supported.
Scrolling updates the deep link to the top visible component using shallow URL
replacement, preserving focus and browser history. Tracking pauses during native
anchor navigation so intermediate cards cannot replace its intended destination.
The **#** anchor beside each component heading copies its absolute component URL
for sharing, without changing the current scroll position or URL. It remains a
native link for opening a new tab or using **Copy link address** if clipboard
access is unavailable.

The single sidebar search accepts names, keywords and implementation status.
**Cmd+K** on macOS or **Ctrl+K** elsewhere focuses the field; its shortcut button
reflects the OS. Arrow keys and Enter navigate matches; Escape clears the query.
Searching matches navigation across both statuses and temporarily reveals the
complete gallery. Selecting a result clears search and reapplies the status filters.
`src/lib/gallery/catalog.ts` maps public package exports
to live examples. Related primitives have their own entries but may share a
composition example (for example TableRow is demonstrated inside Table).

Component READMEs are loaded from the design-system package at build time and
rendered inside their component cards as sanitized Markdown with a short collapsed
teaser and a single inline expand/collapse button. Documentation never opens a
separate popover or dialog.
Components without a README render no documentation block or placeholder;
SelectableRow explicitly shares ExpandableGroup's README. Planned primitives
have their own flat entries. Navigation shows status dots with an Implemented/Planned
legend directly below the search field. These two toggle buttons filter both the
sidebar and gallery; Implemented starts selected and Planned starts unselected.
Filter choices are session-local and are never persisted: opening or reloading
starts with Implemented selected and Planned unselected, even for a planned URL.
The URL-selected component is always visible without changing those choices.
Enable either or both statuses; filtered examples stay mounted to preserve edits
and interaction state.
Component sections retain their status tags;
the Roadmap entry retains the broader planning context.

## Live playgrounds

Every component sample has **Preview** (default) and **Code** tabs. Preview uses
the full card width in a vertically resizable viewport. Reserving its initial
height keeps component anchors stable while lazy frames load. The Code tab uses
CodeMirror with Svelte highlighting, line numbers, completion, folding and undo.
It is loaded on first use and stays mounted across tab switches to retain history.
Svelte syntax errors and compiler accessibility warnings appear inline, in the
lint gutter and in a diagnostics panel; this is not full TypeScript type-checking.
Editor selection uses a distinct focus-color tint, readable text and matching
light/dark configuration; active-line styling does not obscure selected code.
If the editor chunk fails to load, an explicit reload action recovers the cached
module failure and warns that local example edits will be reset.

Edit props, children, snippets, state or scoped CSS; changes compile automatically
after a short debounce. **Copy code** copies the current complete example, including
imports and state. **Reset example** restores its original source and runtime state.
Syntax errors keep the last successful render; runtime errors are shown only at
their example. The **Copy code** button is available from either tab.

The code is derived at build time from each existing `<Sample>` in `sections/`,
including its referenced top-level declarations, imports and scoped styles. There
is no separately maintained code listing. Keep sample state declarative and at the
top level; external application state must not be required to run a sample.
New component section files must also be included in `COMPONENT_SECTION_PATTERN`
in `playground/extract-examples.ts`. Dialog/menu samples can set `previewHeight`
on `<Sample>` to give portalled content enough space.
Foundations and roadmap content retain their ordinary gallery rendering.

The Svelte compiler runs in one lazily started worker per preview tab. Examples
near the viewport activate independent, opaque-origin `sandbox="allow-scripts"`
iframes. Their DS/Svelte runtime is bundled locally by Vite and loaded on demand;
no CDN, origin access, persistence or external network requests are needed or
allowed. Parent appearance changes are forwarded to each frame. Demo links do not
navigate away. Requests are cancelled on source changes; old frames/listeners are
released on replacement or page unmount. Scrolling to another component preserves
the source, selected tab and example state. Recompilation starts fresh runtime state.

Supported imports are `svelte`, `@konfidence/design-system/components`, the DS
package root, and the existing UI5 icon side-effect imports. Other static imports
produce an error. Existing Tailwind utilities are available; new arbitrary utility
classes require rebuilding the stylesheet, while scoped CSS can be edited live.
This is a component playground, not a general-purpose project IDE or an execution
service for hostile code; infinite loops can still make an example unresponsive.

## Run it

From the repository root:

```sh
source ./bin/activate-hermit   # provides pnpm, node
pnpm install
pnpm ds-preview:dev
```

Vite will pick a free port near `5174` (5173 is used by
`konfidence-ui`). Open the URL that Vite prints.

## Scripts

Also available at the workspace root as `pnpm ds-preview:<script>`:

- `pnpm dev` — dev server with HMR (`ds-preview:dev`).
- `pnpm build` — static production build via `@sveltejs/adapter-static`.
- `pnpm start` / `pnpm preview` — serve the production build locally.
- `pnpm check` / `pnpm check:svelte` — tsgo + svelte-check.
- `pnpm lint` / `pnpm lint:fix` — oxlint (+ eslint for Svelte templates).
- `pnpm fmt` / `pnpm fmt:check` — oxfmt.
- `pnpm verify` — run all of the above in check-only mode.
- `pnpm test:unit` — check source extraction, independent compilation and import
  boundaries (also included in `verify`).
- `pnpm test:e2e` — exercise catalog navigation/search/documentation, grouped
  row interactions, playground editing/copy/reset/error recovery and sandbox
  isolation in the built preview app.

## Structure

```
src/
├─ app.html                    # client-rendered SvelteKit app template
├─ app.css                     # Tailwind → Skeleton → @custom-variant dark → @konfidence/design-system/styles
├─ lib/
│  ├─ tokens/read-tokens.ts    # variable-name tables for the Foundations sections
│  ├─ playground/             # source extraction, compiler worker and sandbox lifecycle
│  └─ gallery/
│     ├─ catalog.ts, CatalogItem.svelte, catalog-entry-context.ts
│     ├─ CatalogSearch.svelte, Readme.svelte
│     ├─ Section.svelte, Sample.svelte, NotYetImplemented.svelte, ColorSwatch.svelte, RampGrid.svelte
│     └─ sections/             # component demonstrations, foundations and roadmap
└─ routes/
   ├─ +layout.svelte           # top brandbar, color mode selector, skip link
   ├─ +layout.ts               # ssr = false, prerender = true
    └─ +page.svelte             # complete gallery and flat catalog navigation
```

## Conventions

- The app consumes `@konfidence/design-system` via `workspace:*`. It
  imports the same `styles/index.css` entry and applies the same
  `@custom-variant dark` bridge as `apps/konfidence-ui/src/app.css`.
- Stock `ModeWatcher` starts in System mode and stores preferences under
  `mode-watcher-mode`. It manages `data-theme` using the `konfidence` default
  and applies `.dark` for the resolved appearance. This SPA uses `ssr=false`;
  it does not guarantee mode initialization before first paint.
- Foundation swatches (`Colors.svelte`, `Typography.svelte`) reference
  design tokens by name via `var(--…)`, so the browser reads the value
  live from `packages/konfidence-design-system/src/styles/tokens.css`.
  Changes to the DS tokens show up here on the next HMR.
- Add live components to `gallery/catalog.ts` and their examples to `sections/`.
  Keep preview navigation, search aliases and example state in this application,
  not in the shared design-system components. Pending components stay on Roadmap.

## Deployment

Not wired yet. `adapter-static` produces a plain `build/` folder ready
to drop on any static host (S3, Pages, Cloud Foundry static-content
buildpack); wiring will land in a follow-up PR.

## Roadmap

Track the design-system component roadmap in
`packages/konfidence-design-system/README.md`. When new components ship, add their
live examples and catalog entry, and remove their placeholder from Roadmap.
