# Konfidence dashboard — theme sources

This folder holds the CSS that gives the production dashboard its visual
identity. It combines three ingredients:

1. **Tailwind CSS v4** — utility layer, imported first in `app.css`.
2. **Skeleton v5** — CSS presets and Svelte-component styles
   (`@skeletonlabs/skeleton` + `@skeletonlabs/skeleton-svelte`).
3. **Konfidence design tokens, theme, custom components, and buttons** —
   the CSS files in this folder.

## Files in this folder

Every `.css` file here is a **source file, not a build artifact**. Edit
in place. There is no upstream generator to re-run.

| File                      | Purpose                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `tokens.css`              | Design tokens (colour ramps, spacing, typography, motion, radii, semantic tokens for light and dark, density block). Referenced by every other file in this folder.                                                                                                                                                                                                                                                                                      |
| `konfidence.skeleton.css` | Skeleton theme keyed on `[data-theme="konfidence"], [data-theme="konfidence-dark"]`. Overrides Skeleton core's default `:root` `@theme` colour ramps.                                                                                                                                                                                                                                                                                                    |
| `konfidence.custom.css`   | Konfidence-specific components with no Skeleton equivalent: orbit, phases, diff, timelines, charts, strips, status badges, tags, icon chips, etc.                                                                                                                                                                                                                                                                                                        |
| `konfidence.buttons.css`  | Konfidence `.btn`, `.btn--primary`, `.btn--secondary`, `.btn--ghost`, `.btn--danger`, and `:disabled` state. Skeleton's `@utility btn` alone renders as a plain rectangle without the Konfidence signature (gradient fill, amber glow, hover lift), so these rules are imported after Skeleton and override its `.btn` at equal specificity. The `.btn--primary::before` shine sweep is intentionally omitted — the effect is not shipped in production. |
| `app.css`                 | Single CSS entry point imported once from `+layout.svelte`; wires Tailwind, Skeleton, and the four Konfidence CSS files in the correct order and declares the `@custom-variant dark` that maps Tailwind's `dark:` prefix onto `[data-theme="konfidence-dark"]`.                                                                                                                                                                                          |
| `README.md`               | This file.                                                                                                                                                                                                                                                                                                                                                                                                                                               |

## Relationship to the design-system repository

The initial contents of `tokens.css`, `konfidence.skeleton.css`, and
`konfidence.custom.css` were seeded from
`../konfidence-design/konfidence-design-system/design-system/dist/` and
`konfidence.buttons.css` was seeded from the `.btn*` block of
`dist/konfidence.components.css`. That was a one-time bootstrap.
**The files in this folder are now the authoritative source.**

A follow-up will make the design-system repository consume these files
(or vice versa) so both stay in sync from a single source. Until then,
if a token or component needs to change, edit it here.
