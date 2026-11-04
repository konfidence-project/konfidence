import { validateSearchParams } from "runed/kit";
import * as valibot from "valibot";

/**
 * Runed pre-parses raw query values before validation: `true`/`false` become
 * booleans and values starting with `[` or `{` become JSON. Text and ID fields
 * turn those back into text. JSON whitespace can be normalized by Runed.
 */
const toText = (value: unknown): string =>
  typeof value === "string" ? value : (JSON.stringify(value) ?? "");

/** Free text or a resource ID; missing means unset. */
const urlTextSchema = valibot.optional(
  valibot.pipe(valibot.unknown(), valibot.transform(toText)),
  "",
);

/**
 * Validates single-valued URL filters in a load function.
 *
 * Only the schema's own keys reach Runed. Otherwise an unrelated `_data` param
 * switches Runed to compressed mode, which ignores every other param. Reading
 * each key with `getAll` keeps SvelteKit's per-param load tracking. The last
 * occurrence of a repeated key wins, matching `useSearchParams` on the client.
 */
const validateUrlFilters = <const Entries extends valibot.ObjectEntries>(
  url: URL,
  schema: valibot.ObjectSchema<Entries, undefined>,
): valibot.InferOutput<valibot.ObjectSchema<Entries, undefined>> => {
  const filters = new URL("http://localhost/");
  for (const key of Object.keys(schema.entries)) {
    const value = url.searchParams.getAll(key).at(-1);
    if (value !== undefined) {
      filters.searchParams.set(key, value);
    }
  }
  return validateSearchParams(filters, schema).data;
};

export { urlTextSchema, validateUrlFilters };
