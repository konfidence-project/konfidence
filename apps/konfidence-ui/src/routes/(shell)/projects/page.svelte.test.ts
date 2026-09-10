import { page } from "vitest/browser";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import ProjectsPageTestProvider from "./__tests__/ProjectsPageTestProvider.svelte";

const navigation = vi.hoisted(() => ({
  goto: vi.fn<(url: string, options: { replaceState: boolean }) => Promise<void>>(),
}));

vi.mock("$app/navigation", () => ({ goto: navigation.goto }));
vi.mock("$app/state", () => ({
  page: { url: new URL("http://127.0.0.1/projects") },
}));

beforeEach(() => {
  navigation.goto.mockReset();
});

describe("projects page", () => {
  it("shows feedback while opening the only project", async () => {
    let finishNavigation = (): void => undefined;
    const pendingNavigation = new Promise<void>((resolve) => {
      finishNavigation = resolve;
    });
    navigation.goto.mockReturnValue(pendingNavigation);

    render(ProjectsPageTestProvider, {
      projects: [{ id: "payments", name: "Payments" }],
    });

    await expect.element(page.getByRole("status")).toHaveTextContent("Opening Payments…");
    expect(navigation.goto).toHaveBeenCalledWith("/projects/payments/landscape", {
      replaceState: true,
    });

    finishNavigation();
    await pendingNavigation;
  });
});
