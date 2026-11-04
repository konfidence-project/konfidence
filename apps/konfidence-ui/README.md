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

## Promotions

`/projects/<id>/promotions` lists all promotion configurations and their retained
history. Newest currently means highest sequence, not creation time. Headers
describe the current configuration; instance filters and details use saved
execution references. Succeeded confirms a stage update, not rollout completion.

Groups start collapsed with only the flow header, landscape and latest-waiting
chips visible. Expansion reveals all matching promotions once in sequence order,
even for single-promotion groups. Valid deep links reveal the selected row; later
collapse does not close its details. Selecting the active promotion again closes
the panel; selecting another switches its contents. Clear filters is hidden until a dropdown or
nonblank search is active.

Vector previews use the final repository component with its tag/digest instead
of the registry path, and ellipsize long labels on one line. Hover titles and
instance details retain the complete reference; search still uses the original
value.

Rows show read-only gate progress from the API's declared gate states. Details
list gate types, states and available metadata. Pending explicit approval is
highlighted, not executable. Unknown gate types retain their full identifiers;
missing data does not imply satisfied gates. No creation ages or initiator
identities are fabricated from gate records.

Filters use `promotionLandscape`, `promotionStage` (a JSON `[landscapeId, stageName]`
pair), and `promotionConfig`. `promotion` deep-links to an instance. Search and
independent expansion stay local to the page; embedded mode is preserved.

The page refreshes its existing project-scoped query every ten seconds while
visible and immediately on tab return. Background errors retain content and offer
Retry. A successful response confirming removal closes the selected instance.
This page-local lifecycle is temporary pending the shared refreshing mechanism.

Approve and Reject are disabled placeholders and send no mutation requests.
Eligibility uses the current API: Waiting plus a Pending
`explicit-approval.konfidence.cloud` gate. The plan's legacy `requireApproval` and
approval fields were replaced by gates; recorded gate identity is shown when
available. No backend or API schema changes are part of this view.
