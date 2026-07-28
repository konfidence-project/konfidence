import type { AuthSession, AuthUser } from "$lib/auth/types";
import type { Locale } from "$lib/locale";

declare global {
  namespace App {
    // Interface Error {}
    interface Locals {
      locale: Locale;
      session?: AuthSession;
      user?: AuthUser;
    }
    // Interface PageData {}
    // Interface PageState {}
    // Interface Platform {}
  }
}

// oxlint-disable-next-line unicorn/require-module-specifiers -- SvelteKit requires this file to be an external module.
export {};
