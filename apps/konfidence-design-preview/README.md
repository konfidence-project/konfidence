# `konfidence-design-preview`

Single-page interactive explorer for the Konfidence design system.
The component catalog opens directly at `/`. Mirrors the section layout of the reference
`konfidence-design/konfidence-design-system/design-system.html` and
renders live components from `@konfidence/design-system` alongside
placeholder cards for primitives that have not yet been ported into
the Svelte package.

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
- `pnpm test:e2e` — exercise mode selection, OS changes, persistence,
  and checked menu items in the built preview app.

## Structure

```
src/
├─ app.html                    # client-rendered SvelteKit app template
├─ app.css                     # Tailwind → Skeleton → @custom-variant dark → @konfidence/design-system/styles
├─ lib/
│  ├─ tokens/read-tokens.ts    # variable-name tables for the Foundations sections
│  └─ gallery/
│     ├─ Section.svelte, Sample.svelte, NotYetImplemented.svelte, ColorSwatch.svelte, RampGrid.svelte
│     └─ sections/             # one file per section of the reference gallery
└─ routes/
   ├─ +layout.svelte           # top brandbar, color mode selector, skip link
   ├─ +layout.ts               # ssr = false, prerender = true
   └─ +page.svelte             # component catalog and section navigation
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
- Sections whose Svelte component is not yet available in
  `@konfidence/design-system` render a `<NotYetImplemented>` card that
  lists the planned primitives. Follow-up PRs replace the placeholder
  with real components without touching the section's outer scaffold.

## Deployment

Not wired yet. `adapter-static` produces a plain `build/` folder ready
to drop on any static host (S3, Pages, Cloud Foundry static-content
buildpack); wiring will land in a follow-up PR.

## Roadmap

Track the design-system component roadmap in
`packages/konfidence-design-system/README.md`. When new components ship,
replace the corresponding `<NotYetImplemented />` block in
`src/lib/gallery/sections/<Section>.svelte` with the real component.
