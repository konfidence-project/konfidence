import * as valibot from "valibot";
import { defineEnvVars } from "@sveltejs/kit/hooks";

const apiUrlSchema = valibot.pipe(
  valibot.optional(valibot.string(), "http://localhost:8090"),
  valibot.url("KONFIDENCE_API_URL must be a valid URL"),
  valibot.transform((value) => value.replace(/\/$/, "")),
);

// oxlint-disable-next-line import/prefer-default-export -- SvelteKit expects this named export.
export const variables = defineEnvVars({
  KONFIDENCE_API_URL: {
    description: "Origin of the Konfidence API, including its port when required",
    schema: apiUrlSchema,
  },
});
