import { describe, expect, it, vi } from "vitest";
import type { ApiClient } from "$lib/konfidence-api/client";
import { ProjectsStore } from "$lib/projects/projects.svelte";

const response = (status = 200): Response => new Response(undefined, { status });

describe("project discovery", () => {
  it("loads projects and deduplicates concurrent requests", async () => {
    const get = vi.fn(async () => ({
      data: { data: [{ id: "payments", name: "Payments" }] },
      response: response(),
    }));
    const store = new ProjectsStore({ GET: get } as unknown as ApiClient);

    await Promise.all([store.refresh(), store.refresh()]);

    expect(get).toHaveBeenCalledTimes(1);
    expect(store.status).toBe("ready");
    expect(store.projects).toEqual([{ id: "payments", name: "Payments" }]);
  });

  it("keeps API and network failures retryable", async () => {
    const get = vi
      .fn()
      .mockResolvedValueOnce({ error: {}, response: response(503) })
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ data: { data: [] }, response: response() });
    const store = new ProjectsStore({ GET: get } as unknown as ApiClient);

    await store.refresh();
    expect(store.status).toBe("error");
    expect(store.error).toContain("503");
    await store.refresh();
    expect(store.status).toBe("error");
    expect(store.error).toBe("offline");
    await store.refresh();
    expect(store.status).toBe("ready");
    expect(store.error).toBeUndefined();
  });
});
