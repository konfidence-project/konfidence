# Konfidence Dashboard

The production Konfidence dashboard is a SvelteKit application. Development commands are documented in the repository root [`README.md`](../../README.md#dashboard-development).

## Embedded mode

The dashboard normally renders inside its own application shell (branding, primary navigation, user menu). When embedded into a host application the shell chrome can be hidden while everything else — authentication, routing, project context, page functionality — keeps running.

Trigger embedded mode by adding `?embedded=1` to the URL:

```
https://<host>/projects/<id>/landscape?embedded=1
```

The flag is preserved across internal client-side navigation, so subsequent `<a>` clicks and `goto()` calls stay embedded. Pages use the full viewport in embedded mode; they must not depend on shell-specific spacing.
