import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

/**
 * Demo destination that intentionally raises a 500 so the shared `+error.svelte`
 * boundary renders. Used to eyeball the error path from the primary nav.
 */
export const load: PageLoad = () => {
  error(500, "Demo error triggered from the primary navigation.");
};
