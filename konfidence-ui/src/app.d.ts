// See https://svelte.dev/docs/kit/types#app.d.ts
// For information about these interfaces
import type { SessionUser } from "$lib/server/session";

declare global {
  namespace App {
    // Interface Error {}
    interface Locals {
      user: SessionUser | undefined;
    }
    interface PageData {
      user?: SessionUser;
    }
    // Interface PageState {}
    // Interface Platform {}
  }
}

// oxlint-disable-next-line unicorn/require-module-specifiers -- SvelteKit requires this file to be an external module.
export {};
