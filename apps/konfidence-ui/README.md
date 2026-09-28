# Konfidence Dashboard

The production Konfidence dashboard is a SvelteKit application. Development commands are documented in the repository root [`README.md`](../../README.md#dashboard-development).

## Color mode

The topbar offers Light, Dark, and System. The app mounts stock
`ModeWatcher` in its root layout with `defaultTheme="konfidence"`.
It uses the library defaults: System on first visit and localStorage
`mode-watcher-mode` for preference. The library applies `.dark` for resolved
appearance and manages `data-theme`. The selector does not own storage or
system listeners. This app is client-rendered (`ssr=false`), so the component's
head initialization is not guaranteed to run before first paint in the static
SPA fallback. Storage access failures follow upstream behavior.

## Embedded mode

The dashboard normally renders inside its own application shell (branding, primary navigation, user menu). When embedded into a host application the shell chrome can be hidden while everything else — authentication, routing, project context, page functionality — keeps running.

Trigger embedded mode by adding `?embedded=1` to the URL:

```
https://<host>/projects/<id>/landscape?embedded=1
```

The flag is preserved across internal client-side navigation, so subsequent `<a>` clicks and `goto()` calls stay embedded. Pages use the full viewport in embedded mode; they must not depend on shell-specific spacing.
