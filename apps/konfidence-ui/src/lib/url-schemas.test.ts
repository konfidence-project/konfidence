import { validateSearchParams } from "runed/kit";
import * as valibot from "valibot";
import { describe, expect, it } from "vitest";
import { urlTextSchema } from "#lib/url-schemas.ts";

const schema = valibot.object({ deployment: urlTextSchema });
const url = (query: string): URL => new URL(`http://localhost/view?${query}`);

describe("URL text schema through Runed", () => {
  it("defaults missing fields without changing unrelated URL parameters", () => {
    const current = url("embedded=1&other=keep");
    expect(validateSearchParams(current, schema).data).toEqual({ deployment: "" });
    expect(current.search).toBe("?embedded=1&other=keep");
  });

  it.each(["true", "false", "001", "[]", "{}", "a&b+c d", "ünïcödé", "[unbalanced"])(
    "reads resource ID %j as text",
    (id) => {
      const current = url(new URLSearchParams({ deployment: id }).toString());
      expect(validateSearchParams(current, schema).data?.deployment).toBe(id);
    },
  );

  it("documents Runed's normalization of JSON-shaped IDs", () => {
    expect(validateSearchParams(url("deployment=%5B1%2C%202%5D"), schema).data?.deployment).toBe(
      "[1,2]",
    );
  });
});
