import { describe, expect, it, vi } from "vitest";
import { createApiClient } from "$lib/konfidence-api/client";
import { LandscapeDataStore } from "$lib/landscape/landscape-data.svelte";

describe("landscape requests", () => {
  it("retries both resources after an API failure", async () => {
    let failed = true;
    const fetch = vi.fn(async () =>
      failed
        ? Response.json({ error: { code: "503", message: "Unavailable" } }, { status: 503 })
        : Response.json({ data: [] }),
    );
    const store = new LandscapeDataStore(createApiClient({ fetch }));
    await store.load("payments");
    expect(store.status).toBe("error");
    expect(store.errorStatus).toBe(503);
    failed = false;
    await store.load("payments");
    expect(store.status).toBe("ready");
    expect(store.error).toBeUndefined();
    expect(fetch).toHaveBeenCalledTimes(4);
  });

  it("ignores an old project response even if fetch does not honor cancellation", async () => {
    const pending: (() => void)[] = [];
    const fetch = vi.fn(async (request: Request) => {
      const old = request.url.includes("/old/");
      if (old) {
        await new Promise<void>((resolve) => pending.push(resolve));
      }
      return Response.json({
        data: request.url.endsWith("/landscapes")
          ? [{ id: old ? "old" : "new", name: "Landscape" }]
          : [],
      });
    });
    const store = new LandscapeDataStore(createApiClient({ fetch }));
    const oldRequest = store.load("old");
    await vi.waitFor(() => expect(pending).toHaveLength(2));
    await store.load("new");
    for (const resolve of pending) {
      resolve();
    }
    await oldRequest;
    expect(store.landscapes[0]?.id).toBe("new");
    expect(store.status).toBe("ready");
  });
});
