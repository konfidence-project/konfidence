import { describe, expect, it, vi } from "vitest";
import { createApiClient } from "$lib/konfidence-api/client";
import { LandscapeDataStore } from "$lib/landscape/landscapeData.svelte";

describe("landscape requests", () => {
  it("retries both resources after an API failure", async () => {
    let failed = true;
    const fetch = vi.fn(async () =>
      failed
        ? Response.json({ error: { code: "503", message: "Unavailable" } }, { status: 503 })
        : Response.json({ data: [] }),
    );
    const store = new LandscapeDataStore(createApiClient({ fetch }));
    await store.refresh("payments");
    expect(store.status).toBe("error");
    expect(store.errorStatus).toBe(503);
    expect(store.errorSource).toBe("landscapes");
    failed = false;
    await store.refresh("payments");
    expect(store.status).toBe("ready");
    expect(store.error).toBeUndefined();
    // 1 failed landscapes call, then 1 successful landscapes + 1 successful stages call.
    expect(fetch).toHaveBeenCalledTimes(3);
  });

  it("reports the errorSource as 'stages' when landscapes load but stages fail", async () => {
    const fetch = vi.fn(async (request: Request) =>
      request.url.endsWith("/landscapes")
        ? Response.json({ data: [{ id: "primary", name: "Primary" }] })
        : Response.json({ error: { code: "404", message: "Not found" } }, { status: 404 }),
    );
    const store = new LandscapeDataStore(createApiClient({ fetch }));
    await store.refresh("payments", { landscapeId: "nowhere" });
    expect(store.status).toBe("error");
    expect(store.errorStatus).toBe(404);
    expect(store.errorSource).toBe("stages");
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
    const oldRequest = store.refresh("old");
    await vi.waitFor(() => expect(pending).toHaveLength(1));
    await store.refresh("new");
    for (const resolve of pending) {
      resolve();
    }
    await oldRequest;
    expect(store.landscapes[0]?.id).toBe("new");
    expect(store.status).toBe("ready");
  });
});
