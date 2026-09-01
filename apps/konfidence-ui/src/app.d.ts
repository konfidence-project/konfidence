// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
  namespace App {
    // interface Error {}
    // interface Locals {}
    // interface PageData {}
    // interface PageState {}
    // interface Platform {}
  }

  interface ImportMetaEnv {
    /**
     * Base URL of the Konfidence API. Defaults to `/api` (same-origin proxy).
     * Set to a fully-qualified `https://...` URL when the API is hosted on a
     * different origin; the fetch client then switches to `credentials: "include"`
     * and the backend must send matching CORS headers.
     */
    readonly VITE_KONFIDENCE_API_BASE_URL?: string;
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}

export {};
