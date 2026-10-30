import "../../app.css";
import { page } from "vitest/browser";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import type { ArtifactDeployment, Landscape, Stage } from "#lib/konfidence-api/types.ts";
import ArtifactDeploymentsView from "./ArtifactDeploymentsView.svelte";

const navigation = vi.hoisted(() => ({
  goto: vi.fn<(url: URL, options?: Record<string, unknown>) => Promise<void>>(),
}));
const appState = vi.hoisted(() => ({
  page: {
    state: {} as App.PageState,
    url: new URL("http://127.0.0.1/projects/payments/artifact-deployments"),
  },
}));

vi.mock("$app/navigation", () => ({ goto: navigation.goto }));
vi.mock("$app/state", () => appState);

const landscapes: Landscape[] = [{ id: "development", name: "Development" }];
const stages: Stage[] = [{ id: "dev", landscapeId: "development", name: "Development" }];
const deployment = (id: string, version: string): ArtifactDeployment => ({
  artifact: {
    componentName: "payments-api",
    componentVersion: version,
    repository: "registry.example",
  },
  id,
  landscapeId: "development",
  stageIds: ["dev"],
  status: "ArtifactDeployed",
  vectorDeploymentIds: [],
});
const props = {
  artifactDeployments: [deployment("artifact-one", "1.0.0"), deployment("artifact-two", "2.0.0")],
  error: undefined,
  hasLoaded: true,
  landscapes,
  loading: false,
  stages,
  vectorDeployments: [],
};

const setDetail = (id: string, origin?: App.PageState["overlayOrigin"]): void => {
  appState.page.url = new URL(
    `http://127.0.0.1/projects/payments/artifact-deployments?deployment=${id}`,
  );
  appState.page.state = origin ? { overlayOrigin: origin } : {};
};

beforeEach(() => {
  navigation.goto.mockReset();
  navigation.goto.mockResolvedValue();
  appState.page.url = new URL("http://127.0.0.1/projects/payments/artifact-deployments");
  appState.page.state = {};
});

describe("<ArtifactDeploymentsView> detail panel", () => {
  it("is non-modal on desktop and closes when an outside control is clicked", async () => {
    await page.viewport(1200, 800);
    setDetail("artifact-one");
    await render(ArtifactDeploymentsView, props);
    const dialog = page.getByRole("dialog", { name: "payments-api@1.0.0" });
    await expect.element(dialog).toHaveAttribute("aria-modal", "false");

    const searches = document.querySelectorAll<HTMLInputElement>("[data-testid='artifact-search']");
    const search = searches.item(searches.length - 1);
    search.focus();
    search.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0 }));

    await vi.waitFor(() => expect(navigation.goto).toHaveBeenCalledOnce());
    const [next, options] = navigation.goto.mock.calls[0]!;
    expect(next.searchParams.has("deployment")).toBe(false);
    expect(options?.replace).toBe(true);
    expect(document.activeElement).toBe(search);
  });

  it("switches rows by replacing the active overlay history entry", async () => {
    await page.viewport(1200, 800);
    setDetail("artifact-one", "deployment");
    await render(ArtifactDeploymentsView, props);

    const row = document.querySelector<HTMLElement>("[data-row-id='artifact-two']")!;
    row.click();

    await vi.waitFor(() => expect(navigation.goto).toHaveBeenCalledOnce());
    const [next, options] = navigation.goto.mock.calls[0]!;
    expect(next.searchParams.get("deployment")).toBe("artifact-two");
    expect(options).toMatchObject({ replace: true, state: { overlayOrigin: "deployment" } });
  });

  it("remains modal on mobile", async () => {
    await page.viewport(390, 700);
    expect(globalThis.innerWidth).toBe(390);
    setDetail("artifact-one");
    await render(ArtifactDeploymentsView, props);
    const dialog = page.getByRole("dialog", { name: "payments-api@1.0.0" });
    await expect.element(dialog).toBeVisible();
    await expect.element(dialog).toHaveAttribute("aria-modal", "true");
  });

  it("closes when a client filter hides the selected row", async () => {
    await page.viewport(1200, 800);
    const back = vi.spyOn(globalThis.history, "back").mockImplementation(() => undefined);
    setDetail("artifact-one", "deployment");
    await render(ArtifactDeploymentsView, props);
    const search = page.getByTestId("artifact-search").element() as HTMLInputElement;

    search.value = "artifact-two";
    search.dispatchEvent(new InputEvent("input", { bubbles: true }));

    await vi.waitFor(() => expect(back).toHaveBeenCalledOnce());
    back.mockRestore();
  });
});
