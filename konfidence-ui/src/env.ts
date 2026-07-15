import { defineEnvVars } from "@sveltejs/kit/hooks";
import * as v from "valibot";

export const variables = defineEnvVars({
  KONFIDENCE_API_URL: {
    description: "Base URL of the Konfidence API, including its port when required",
    schema: v.pipe(
      v.optional(v.string(), "http://localhost:8090"),
      v.url("KONFIDENCE_API_URL must be a valid URL"),
      v.transform((url) => url.replace(/\/$/, "")),
    ),
  },
});
