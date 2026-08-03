import type { AuthSession, AuthUser } from "$lib/auth/types";

declare global {
  namespace App {
    interface Error {
      code?: string;
      message: string;
    }
    interface Locals {
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
