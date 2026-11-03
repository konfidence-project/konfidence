import "../../../app.css";
import { page } from "vitest/browser";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render } from "vitest-browser-svelte";
import type {
  Landscape,
  Stage,
  VectorDeployment,
  VectorPromotionConfig,
} from "#lib/konfidence-api/types.ts";
import LandscapeOverview from "#lib/landscape/components/LandscapeOverview.svelte";

const navigation = vi.hoisted(() => ({
  goto: vi.fn<(url: URL | string, options?: Record<string, unknown>) => Promise<void>>(),
  pushState: vi.fn<(url: string | URL, state: App.PageState) => void>(),
  replaceState: vi.fn<(url: string | URL, state: App.PageState) => void>(),
}));
const appState = vi.hoisted(() => ({
  page: {
    state: {} as App.PageState,
    url: new URL("http://127.0.0.1/projects/payments/landscape"),
  },
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

vi.mock("#lib/queries.svelte.ts", () => ({
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

vi.mock("$app/navigation", () => ({
  goto: navigation.goto,
  pushState: navigation.pushState,
  replaceState: navigation.replaceState,
}));
vi.mock("$app/state", () => appState);

const setUrl = (search: string): void => {
  appState.page.url = new URL(`http://127.0.0.1/projects/payments/landscape${search}`);
};

const setStageOverlay = (landscapeId: string, stageId: string): void => {
  appState.page.state = {
    overlayOrigin: "landscape",
    stageOverlay: { landscapeId, projectId: "payments", stageId },
  };
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
  navigation.pushState.mockReset();
  navigation.replaceState.mockReset();
  vectorDeployment.states.clear();
  vectorDeployment.reload.mockReset();
  appState.page.state = {};
  setUrl("");
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("<LandscapeOverview> stage overlay", () => {
  it("pre-opens the overlay from shallow route state", async () => {
    setStageOverlay("primary", "dev-api");
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByRole("dialog", { name: "dev-api" })).toBeVisible();
  });

  it("keeps the overlay closed with no shallow route state", async () => {
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders stage details for the shallow route state", async () => {
    setStageOverlay("primary", "dev-api");
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByRole("dialog", { name: "dev-api" })).toBeVisible();
    await expect.element(page.getByRole("region", { name: "Target version" })).toBeVisible();
    await expect.element(page.getByRole("region", { name: "Active version" })).toBeVisible();
    await expect.element(page.getByText("Deploying vector", { exact: true })).toBeVisible();
    await expect.element(page.getByText("Ready", { exact: true })).toBeVisible();
  });

  it("highlights the selected stage's card when its overlay is open", async () => {
    setStageOverlay("primary", "dev-api");
    await render(LandscapeOverview, baseProps);

    const selectedCard = page
      .getByRole("link", { name: "View details for stage dev-api in Primary" })
      .element() as HTMLElement;
    await vi.waitFor(() => {
      expect(selectedCard.classList.contains("stage-card--selected")).toBe(true);
    });
  });

  it("does not open the overlay for an unknown stage reference", async () => {
    setStageOverlay("primary", "missing");
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
  });

  it("navigates back once when the overlay is closed", async () => {
    const back = vi.spyOn(globalThis.history, "back").mockImplementation(() => undefined);
    setStageOverlay("primary", "dev-api");
    await render(LandscapeOverview, baseProps);

    await page.getByTestId("side-panel-close").click();

    expect(back).toHaveBeenCalledOnce();
  });

  it("shallow-routes an ordinary stage-card click to its canonical URL", async () => {
    await render(LandscapeOverview, baseProps);

    const card = page.getByRole("link", { name: "View details for stage dev-api in Primary" });
    card
      .element()
      .dispatchEvent(new MouseEvent("click", { bubbles: true, button: 0, cancelable: true }));

    expect(navigation.pushState).toHaveBeenCalledWith(
      "/projects/payments/landscape/primary/stages/dev-api",
      {
        overlayOrigin: "landscape",
        stageOverlay: {
          landscapeId: "primary",
          projectId: "payments",
          stageId: "dev-api",
        },
      },
    );
  });

  it("does not intercept a modified stage-card click", async () => {
    await render(LandscapeOverview, baseProps);

    const card = page
      .getByRole("link", { name: "View details for stage dev-api in Primary" })
      .element();
    card.addEventListener("click", (event) => event.preventDefault());
    card.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, ctrlKey: true }));

    expect(navigation.pushState).not.toHaveBeenCalled();
  });

  it("preserves the embedded flag in the canonical card URL", async () => {
    await render(LandscapeOverview, { ...baseProps, embedded: true });

    await expect
      .element(page.getByRole("link", { name: "View details for stage dev-api in Primary" }))
      .toHaveAttribute("href", "/projects/payments/landscape/primary/stages/dev-api?embedded=1");
  });

  it("links from the overlay to the canonical full page", async () => {
    setStageOverlay("primary", "dev-api");
    await render(LandscapeOverview, baseProps);

    await expect
      .element(page.getByRole("link", { name: "Open full page" }))
      .toHaveAttribute("href", "/projects/payments/landscape/primary/stages/dev-api");
  });

  it("selects a stage by landscape and stage id", async () => {
    const secondaryStage = { ...stage, landscapeId: "secondary" };
    setStageOverlay("secondary", "dev-api");
    await render(LandscapeOverview, {
      ...baseProps,
      landscapes: [...landscapes, { id: "secondary", name: "Secondary" }],
      stages: [stage, secondaryStage],
    });

    const primary = page.getByRole("link", {
      name: "View details for stage dev-api in Primary",
    });
    const secondary = page.getByRole("link", {
      name: "View details for stage dev-api in Secondary",
    });
    await vi.waitFor(() => {
      expect((primary.element() as HTMLElement).classList.contains("stage-card--selected")).toBe(
        false,
      );
      expect((secondary.element() as HTMLElement).classList.contains("stage-card--selected")).toBe(
        true,
      );
    });
  });
});

describe("<LandscapeOverview> vector deployment sections", () => {
  it("shows a loading indicator for each section while the vector deployment is fetching", async () => {
    setStageOverlay("primary", "dev-api");
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
    setStageOverlay("primary", "dev-api");
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
    setStageOverlay("primary", "dev-api");
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
    setStageOverlay("primary", "dev-api");
    await render(LandscapeOverview, baseProps);

    await expect.element(page.getByText("No vector deployment")).toBeVisible();
    await expect.element(page.getByRole("alert")).not.toBeInTheDocument();
  });

  it("collapses to a single vector deployment section when target and active share an id", async () => {
    setVectorDeploymentState("dev-api-v1", {
      data: vectorDeploymentResource({ id: "dev-api-v1", status: "DeploymentReady" }),
    });
    setStageOverlay("primary", "prod-api");
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

describe("<LandscapeOverview> promotion overlay", () => {
  const promotionConfig: VectorPromotionConfig = {
    id: "dev-to-prod",
    promotions: [
      {
        id: "promo-1",
        source: { kind: "Stage", landscape: "primary", name: "dev-api" },
        status: "InProgress",
        target: { kind: "Stage", landscape: "primary", name: "prod-api" },
        vector: "registry.example.com/delivery/vector:2.0.0",
      },
    ],
    source: { kind: "Stage", landscape: "primary", name: "dev-api" },
    target: { kind: "Stage", landscape: "primary", name: "prod-api" },
  };

  const promotionProps = {
    ...baseProps,
    promotionConfigs: [promotionConfig],
    stages: [stage, rolledOutStage],
  };

  const promotionId = "dev-to-prod";

  it("keeps the bottom sheet closed with no promotion param", async () => {
    await render(LandscapeOverview, promotionProps);

    await expect.element(page.getByTestId("bottom-sheet")).not.toBeVisible();
  });

  it("opens the bottom sheet titled source → target for the selected promotion", async () => {
    setUrl(`?promotion=${encodeURIComponent(promotionId)}`);
    await render(LandscapeOverview, promotionProps);

    await expect.element(page.getByRole("dialog", { name: "dev-api → prod-api" })).toBeVisible();
    const details = page.getByTestId("promotion-details");
    await expect.element(details).toBeVisible();
    await expect.element(page.getByRole("heading", { name: "Artifact Changes" })).toBeVisible();
    await expect.element(page.getByRole("heading", { name: "Feature Toggles" })).toBeVisible();
    await expect.element(page.getByRole("heading", { name: "Info" })).toBeVisible();
  });

  it("opens promotion details with Enter from an accessible edge button", async () => {
    await page.viewport(1200, 800);
    await render(LandscapeOverview, promotionProps);

    const edge = page.getByRole("button", {
      name: "View promotion details from dev-api to prod-api",
    });
    await expect.element(edge).toBeInTheDocument();
    edge.element().dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "Enter" }));

    await vi.waitFor(() => expect(navigation.goto).toHaveBeenCalledOnce());
    const [next] = navigation.goto.mock.calls[0]!;
    expect(new URL(next).searchParams.get("promotion")).toBe(promotionId);
  });

  it("closes promotion details when blank canvas is clicked but not when it is panned", async () => {
    await page.viewport(1200, 800);
    setUrl(`?promotion=${encodeURIComponent(promotionId)}`);
    await render(LandscapeOverview, promotionProps);

    const pane = document.querySelector<HTMLElement>(".svelte-flow__pane")!;
    pane.dispatchEvent(new MouseEvent("click", { bubbles: true }));

    await vi.waitFor(() => expect(navigation.goto).toHaveBeenCalledOnce());
    const [next] = navigation.goto.mock.calls[0]!;
    expect(new URL(next).searchParams.has("promotion")).toBe(false);
  });

  it("uses a non-modal sheet on desktop and converts it to modal on mobile", async () => {
    await page.viewport(1200, 800);
    setUrl(`?promotion=${encodeURIComponent(promotionId)}`);
    await render(LandscapeOverview, promotionProps);
    const dialog = page.getByTestId("bottom-sheet");
    await expect.element(dialog).toHaveAttribute("aria-modal", "false");

    await page.viewport(390, 700);
    await expect.element(dialog).toHaveAttribute("aria-modal", "true");
    await expect.element(dialog).toBeVisible();
  });

  it("does not open the bottom sheet for an unknown promotion id", async () => {
    setUrl("?promotion=missing");
    await render(LandscapeOverview, promotionProps);

    await expect.element(page.getByTestId("bottom-sheet")).not.toBeVisible();
  });

  it("highlights both stages connected by the selected promotion", async () => {
    setUrl(`?promotion=${encodeURIComponent(promotionId)}`);
    await render(LandscapeOverview, promotionProps);

    const sourceCard = page
      .getByRole("link", { name: "View details for stage dev-api in Primary" })
      .element() as HTMLElement;
    const targetCard = page
      .getByRole("link", { name: "View details for stage prod-api in Primary" })
      .element() as HTMLElement;
    await vi.waitFor(() => {
      expect(sourceCard.classList.contains("stage-card--selected")).toBe(true);
      expect(targetCard.classList.contains("stage-card--selected")).toBe(true);
    });
  });

  it("clears the promotion param when the sheet is closed", async () => {
    setUrl(`?promotion=${encodeURIComponent(promotionId)}`);
    await render(LandscapeOverview, promotionProps);

    const dialog = page.getByRole("dialog", { name: "dev-api → prod-api" });
    await expect.element(dialog).toBeVisible();
    (dialog.getByTestId("bottom-sheet-close").element() as HTMLButtonElement).click();

    expect(navigation.goto).toHaveBeenCalled();
    const [next, options] = navigation.goto.mock.calls[0]!;
    expect(new URL(next).searchParams.has("promotion")).toBe(false);
    expect(options).toEqual({ replace: true, reset: false });
  });

  it("keeps the stage and promotion panels mutually exclusive", async () => {
    setUrl(`?promotion=${encodeURIComponent(promotionId)}`);
    await render(LandscapeOverview, promotionProps);

    await expect.element(page.getByTestId("bottom-sheet")).toBeVisible();
    // The stage (right) panel must not be open while a promotion is selected.
    await expect.element(page.getByTestId("side-panel")).not.toBeVisible();
  });

  it("finishes closing the stage panel before selecting a promotion", async () => {
    await page.viewport(1200, 800);
    setStageOverlay("primary", "dev-api");
    await render(LandscapeOverview, promotionProps);
    let edge: Element | null = null;
    await vi.waitFor(() => {
      edge = document.querySelector(".svelte-flow__edge[data-id='dev-to-prod']");
      expect(edge).not.toBeNull();
    });

    edge!.dispatchEvent(new MouseEvent("click", { bubbles: true, button: 0 }));
    expect(navigation.goto).not.toHaveBeenCalled();
    const sidePanel = page.getByTestId("side-panel").element();
    await expect.element(sidePanel).toHaveAttribute("data-state", "closed");
    expect(navigation.goto).not.toHaveBeenCalled();
    sidePanel.dispatchEvent(
      new TransitionEvent("transitionend", { bubbles: true, propertyName: "translate" }),
    );

    await vi.waitFor(() => expect(navigation.goto).toHaveBeenCalledOnce(), { timeout: 1000 });
    const [next, options] = navigation.goto.mock.calls[0]!;
    expect(new URL(next).searchParams.get("promotion")).toBe(promotionId);
    expect(options?.replace).toBe(true);
  });

  it("finishes closing promotion details before selecting a stage", async () => {
    await page.viewport(1200, 800);
    setUrl(`?promotion=${encodeURIComponent(promotionId)}`);
    appState.page.state = { overlayOrigin: "landscape" };
    await render(LandscapeOverview, promotionProps);
    const stageCard = page.getByRole("link", {
      name: "View details for stage dev-api in Primary",
    });

    stageCard
      .element()
      .dispatchEvent(new MouseEvent("click", { bubbles: true, button: 0, cancelable: true }));
    expect(navigation.replaceState).not.toHaveBeenCalled();
    const bottomSheet = page.getByTestId("bottom-sheet").element();
    await expect.element(bottomSheet).toHaveAttribute("data-state", "closed");
    expect(navigation.replaceState).not.toHaveBeenCalled();
    bottomSheet.dispatchEvent(
      new TransitionEvent("transitionend", { bubbles: true, propertyName: "translate" }),
    );

    await vi.waitFor(() => expect(navigation.replaceState).toHaveBeenCalledOnce(), {
      timeout: 1000,
    });
    expect(navigation.replaceState.mock.calls[0]![0]).toBe(
      "/projects/payments/landscape/primary/stages/dev-api",
    );
  });
});
