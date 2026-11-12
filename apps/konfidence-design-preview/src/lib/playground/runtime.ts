// A self-contained browser bundle: the opaque-origin sandbox cannot import
// workspace packages, access the application's origin, or contact a CDN.
export * as client from "svelte/internal/client";
export * as components from "@konfidence/design-system/components";
export * as svelte from "svelte";
