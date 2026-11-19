// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
  namespace App {
    // interface Error {}
    // interface Locals {}
    // interface PageData {}
    interface PageState {
      readonly overlayOrigin?: "landscape" | "deployment" | "promotion";
      readonly stageOverlay?: {
        readonly projectId: string;
        readonly landscapeId: string;
        readonly stageId: string;
      };
    }
    // interface Platform {}
  }

  interface ImportMetaEnv {
    /**
     * Base URL of the Konfidence API server without the `/api` path or a
     * trailing slash. Defaults to the current origin. Set to a fully-qualified
     * URL when the API is hosted on a different origin.
     */
    readonly VITE_KONFIDENCE_API_BASE_URL?: string;
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}

export {};
