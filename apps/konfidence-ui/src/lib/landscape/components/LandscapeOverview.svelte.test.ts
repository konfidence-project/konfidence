import "../../../app.css";
import { page } from "vitest/browser";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import type {
  Landscape,
  Stage,
  VectorDeployment,
  VectorPromotionConfig,
} from "$lib/konfidence-api/types";
import LandscapeOverview from "$lib/landscape/components/LandscapeOverview.svelte";

const navigation = vi.hoisted(() => ({
  goto: vi.fn<(url: URL, options: { keepFocus: boolean; noScroll: boolean }) => Promise<void>>(),
}));
const appState = vi.hoisted(() => ({
  page: { url: new URL("http://127.0.0.1/projects/payments/landscape") },
}));

interface MockQueryState {
  loading: boolean;
  data: VectorDeployment | undefined;
  error: { message: string; status?: number } | undefined;
  reload: () => void;
}

const vectorDeployment = vi.hoisted(() => ({
  reload: vi.fn(),
  states: new Map<string, MockQueryState>(),
}));

vi.mock("$lib/queries.svelte", () => ({
  useVectorDeployment: (paramGetter: () => { vectorDeploymentId: string }): MockQueryState => {
    const { vectorDeploymentId } = paramGetter();
    return (
      vectorDeployment.states.get(vectorDeploymentId) ?? {
        data: undefined,
        error: undefined,
        loading: true,
        reload: vectorDeployment.reload,
      }
    );
  },
}));

const setVectorDeploymentState = (id: string, state: Partial<MockQueryState>): void => {
  vectorDeployment.states.set(id, {
    data: undefined,
    error: undefined,
    loading: false,
    reload: vectorDeployment.reload,
    ...state,
  });
};

const vectorDeploymentResource = (overrides: Partial<VectorDeployment>): VectorDeployment => ({
  id: "dev-api-v2",
  landscapeId: "primary",
  stageId: "dev-api",
  status: "DeployingVector",
  vector: {
    componentName: "delivery-service",
    componentVersion: "2.0.0",
    repository: "registry.example.com/delivery/vector",
  },
  ...overrides,
});

vi.mock("$app/navigation", () => ({ goto: navigation.goto }));
vi.mock("$app/state", () => appState);

const setUrl = (search: string): void => {
  appState.page.url = new URL(`http://127.0.0.1/projects/payments/landscape${search}`);
};

const landscapes: readonly Landscape[] = [{ id: "primary", name: "Primary" }];
const stage: Stage = {
  activeStageVersion: {
    id: "dev-api-v1",
    stageGeneration: 1,
    status: "Ready",
    vector: "registry.example.com/delivery/vector:1.0.0",
  },
  id: "dev-api",
  landscapeId: "primary",
  name: "dev-api",
  targetStageVersion: {
    id: "dev-api-v2",
    stageGeneration: 2,
    status: "DeployingVector",
    vector: "registry.example.com/delivery/vector:2.0.0",
  },
};
const stages: readonly Stage[] = [stage];
const promotionConfigs: readonly VectorPromotionConfig[] = [];

const rolledOutStage: Stage = {
  activeStageVersion: {
    id: "dev-api-v1",
    stageGeneration: 1,
    status: "Ready",
    vector: "registry.example.com/delivery/vector:1.0.0",
  },
  id: "prod-api",
  landscapeId: "primary",
  name: "prod-api",
  targetStageVersion: {
    id: "dev-api-v1",
    stageGeneration: 1,
    status: "Ready",
    vector: "registry.example.com/delivery/vector:1.0.0",
  },
};

const baseProps = {
  embedded: false,
  landscapes,
  onRetry: vi.fn(),
  projectId: "payments",
  promotionConfigs,
  stages,
  status: "ready" as const,
};

beforeEach(() => {
  navigation.goto.mockReset();
  navigation.goto.mockResolvedValue();
  vectorDeployment.states.clear();
  vectorDeployment.reload.mockReset();
  setUrl("");
});

describe("<LandscapeOverview> stage overlay", () => {
  it("keeps the overlay closed with no stage param", async () => {
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
  });

  it("pre-opens the overlay for the stage named in the search param", async () => {
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByRole("dialog", { name: "dev-api" })).toBeVisible();
    await expect.element(page.getByRole("region", { name: "Target version" })).toBeVisible();
    await expect.element(page.getByRole("region", { name: "Active version" })).toBeVisible();
    await expect.element(page.getByText("Deploying vector", { exact: true })).toBeVisible();
    await expect.element(page.getByText("Ready", { exact: true })).toBeVisible();
  });

  it("highlights the selected stage's card when its overlay is open", async () => {
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, baseProps);

    const selectedCard = page
      .getByRole("link", { name: "View details for stage dev-api in Primary" })
      .element() as HTMLElement;
    await vi.waitFor(() => {
      expect(selectedCard.classList.contains("stage-card--selected")).toBe(true);
    });
  });

  it("does not open the overlay for an unknown stage id", async () => {
    setUrl("?stage=missing");
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
  });

  it("clears the stage param when the overlay is closed", async () => {
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, baseProps);

    await page.getByTestId("side-panel-close").click();

    expect(navigation.goto).toHaveBeenCalled();
    const [next, options] = navigation.goto.mock.calls[0]!;
    expect(next.searchParams.has("stage")).toBe(false);
    expect(next.pathname).toBe("/projects/payments/landscape");
    expect(options).toEqual({ keepFocus: true, noScroll: true });
  });

  it("navigates to the dedicated stage detail page from the overlay resize button", async () => {
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, baseProps);

    const dialog = page.getByRole("dialog", { name: "dev-api" });
    await expect.element(dialog).toBeVisible();
    // A direct DOM click avoids a stale top-layer backdrop (left by a prior
    // test's unmounted <dialog>) swallowing the synthesized pointer event.
    (dialog.getByTestId("side-panel-resize").element() as HTMLButtonElement).click();

    expect(navigation.goto).toHaveBeenCalledWith(
      "/projects/payments/landscape/primary/stages/dev-api",
    );
  });

  it("preserves the embedded flag when navigating to the stage detail page", async () => {
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, { ...baseProps, embedded: true });

    const dialog = page.getByRole("dialog", { name: "dev-api" });
    await expect.element(dialog).toBeVisible();
    (dialog.getByTestId("side-panel-resize").element() as HTMLButtonElement).click();

    expect(navigation.goto).toHaveBeenCalledWith(
      "/projects/payments/landscape/primary/stages/dev-api?embedded=1",
    );
  });
});

describe("<LandscapeOverview> vector deployment sections", () => {
  it("shows a loading indicator for each section while the vector deployment is fetching", async () => {
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, baseProps);

    const loaders = page.getByText("Loading vector deployment");
    await expect.element(loaders.first()).toBeVisible();
    await expect.element(loaders.nth(1)).toBeVisible();
    await expect.element(page.getByRole("region", { name: "Target version" })).toBeVisible();
    await expect.element(page.getByRole("region", { name: "Active version" })).toBeVisible();
  });

  it("renders every vector deployment field on success", async () => {
    setVectorDeploymentState("dev-api-v2", {
      data: vectorDeploymentResource({ id: "dev-api-v2" }),
    });
    setVectorDeploymentState("dev-api-v1", {
      data: vectorDeploymentResource({
        id: "dev-api-v1",
        status: "DeploymentReady",
        vector: {
          componentName: "delivery-service",
          componentVersion: "1.0.0",
          repository: "registry.example.com/delivery/vector",
        },
      }),
    });
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByText("delivery-service").first()).toBeVisible();
    await expect.element(page.getByText("2.0.0", { exact: true })).toBeVisible();
    await expect.element(page.getByText("1.0.0", { exact: true })).toBeVisible();
    await expect
      .element(page.getByText("registry.example.com/delivery/vector").first())
      .toBeVisible();
    const ids = page.getByTestId("vectordeployment-detail-id");
    await expect.element(ids.first()).toHaveTextContent("dev-api-v2");
    await expect.element(ids.nth(1)).toHaveTextContent("dev-api-v1");
  });

  it("renders an inline error with a retry action that reloads the section", async () => {
    const reload = vi.fn();
    setVectorDeploymentState("dev-api-v2", {
      error: { message: "Vector deployment is currently unavailable.", status: 503 },
      reload,
    });
    setVectorDeploymentState("dev-api-v1", {
      data: vectorDeploymentResource({ id: "dev-api-v1", status: "DeploymentReady" }),
    });
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, baseProps);

    const alert = page.getByRole("alert").first();
    await expect.element(alert).toMatchTextContent("Vector deployment could not be loaded");
    await expect.element(page.getByRole("region", { name: "Target version" })).toBeVisible();

    await page.getByRole("button", { name: "Retry" }).first().click();
    expect(reload).toHaveBeenCalledOnce();
  });

  it("shows a 'no vector deployment' empty state on 404 instead of an error", async () => {
    setVectorDeploymentState("dev-api-v2", {
      error: { message: "Vector deployment is currently unavailable.", status: 404 },
    });
    setVectorDeploymentState("dev-api-v1", {
      data: vectorDeploymentResource({ id: "dev-api-v1", status: "DeploymentReady" }),
    });
    setUrl("?stage=dev-api");
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByText("No vector deployment")).toBeVisible();
    await expect.element(page.getByRole("alert")).not.toBeInTheDocument();
  });

  it("collapses to a single vector deployment section when target and active share an id", async () => {
    setVectorDeploymentState("dev-api-v1", {
      data: vectorDeploymentResource({ id: "dev-api-v1", status: "DeploymentReady" }),
    });
    setUrl("?stage=prod-api");
    await render(LandscapeOverview, {
      ...baseProps,
      landscapes: [{ id: "primary", name: "Primary" }],
      stages: [rolledOutStage],
    });

    const ids = page.getByTestId("vectordeployment-detail-id");
    await expect.element(ids.first()).toHaveTextContent("dev-api-v1");
    await expect.element(ids.nth(1)).not.toBeInTheDocument();
    await expect.element(page.getByRole("region", { name: "Target version" })).toBeVisible();
    await expect.element(page.getByRole("region", { name: "Active version" })).toBeVisible();
  });
});
