# CLI login pages

SvelteKit prerenders the CLI login success and failure pages with the shared
Konfidence design system. The build writes two self-contained HTML documents
with inline CSS and logos, system-theme initialization, and a best-effort
close-window timer. Client-side rendering is disabled; there is no hydration.
SvelteKit inlines the CSS. Its disabled, zero-width stylesheet link is inert;
the pages fetch no assets at runtime.

## Develop and generate

From the repository root, with the Hermit environment activated:

```sh
pnpm install --frozen-lockfile
pnpm cli-pages:build
```

Inspect `internal/kden/auth/pages/generated/success.html` and `failure.html`
after building. This app does not run a server; only the Go CLI serves these pages.

```sh
pnpm cli-pages:verify
pnpm cli-pages:check:generated
```

The Vite build writes the documents directly to the Go embed directory. Commit
those files with source changes. The CLI embeds them with `go:embed`; ordinary
Go builds can use the committed files without Node. `make build-kden-cli` and
release builds regenerate them first. Frontend CI checks that the committed
files match the Vite build.

The two routes render their content inside a shared page layout with the design
system's Brandbar, inline BrandLogo, and styles. The design system owns a copy
of the original wordmark SVG; the dashboard's existing assets remain
unchanged. The callback uses the system color scheme rather than dashboard storage.

## Browser and screenshot tests

Playwright intercepts navigation and loads the generated documents under the
callback's CSP without starting a server. Tests cover both results in light and
dark mode at desktop width. They also check for asset requests, blocked window
closing, system-theme changes, and rendering without JavaScript.

Generate the four tracked Linux screenshots with Docker:

```sh
pnpm cli-pages:screenshots:generate
pnpm cli-pages:screenshots:check
```

The shared screenshot scripts use the same Linux ARM64 Playwright image as
design-system tests. The check regenerates in isolation and rejects changed,
missing, or obsolete baselines. Review the PNG changes before committing.

For a native browser run:

```sh
pnpm --filter konfidence-cli-pages exec playwright install chromium
pnpm cli-pages:test
```

On macOS, first create local, ignored baselines with
`pnpm cli-pages:test --update-snapshots`. Normal test runs never update baselines.
