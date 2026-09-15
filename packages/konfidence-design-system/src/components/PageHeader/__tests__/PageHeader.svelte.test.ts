import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import { createRawSnippet } from "svelte";

import PageHeader from "../PageHeader.svelte";
import PageHeaderFixture from "./PageHeaderFixture.svelte";
import "../../../../../../apps/konfidence-ui/src/app.css";

describe("<PageHeader>", () => {
  it("renders the title as an h1 with the DS display recipe", () => {
    render(PageHeader, { title: "Landscapes" });
    const heading = document.querySelector('[data-testid="page-heading"]');
    expect(heading).not.toBeNull();
    expect(heading?.tagName).toBe("H1");
    expect(heading?.textContent?.trim()).toBe("Landscapes");
  });

  it("forwards id onto the h1 so callers can wire aria-labelledby", () => {
    const id = "artifact-view-title";
    render(PageHeader, { id, title: "Artifact Deployments" });
    const heading = document.getElementById(id);
    expect(heading?.tagName).toBe("H1");
  });

  it("shows the description under the title when provided", async () => {
    render(PageHeader, {
      description: "Inspect the deployments backing this project.",
      title: "Artifact Deployments",
    });
    await expect
      .element(page.getByText("Inspect the deployments backing this project."))
      .toBeVisible();
  });

  it("renders the eyebrow snippet above the title", () => {
    const eyebrow = createRawSnippet(() => ({
      render: () => '<nav data-testid="eyebrow-content">Landscapes / dev-api</nav>',
    }));
    render(PageHeader, { eyebrow, title: "dev-api" });
    const eye = document.querySelector('[data-testid="eyebrow-content"]');
    const heading = document.querySelector('[data-testid="page-heading"]');
    expect(eye).not.toBeNull();
    expect(heading).not.toBeNull();
    expect(Boolean(eye!.compareDocumentPosition(heading!) & Node.DOCUMENT_POSITION_FOLLOWING)).toBe(
      true,
    );
  });

  it("renders the actions snippet next to the title", () => {
    const actions = createRawSnippet(() => ({
      render: () => '<button data-testid="primary-action">Promote</button>',
    }));
    render(PageHeader, { actions, title: "Overview" });
    expect(document.querySelector('[data-testid="primary-action"]')).not.toBeNull();
  });

  describe("variant screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.removeAttribute("data-mode");
    });

    const eyebrowSnippet = createRawSnippet(() => ({
      render: () =>
        '<nav aria-label="Breadcrumb" style="display:flex;align-items:center;gap:6px;font-size:var(--text-sm);color:var(--text-tertiary);"><a href="#" style="color:var(--text-tertiary);text-decoration:none;">Landscapes</a><span style="color:var(--border-strong);">/</span><a href="#" style="color:var(--text-tertiary);text-decoration:none;">Primary</a><span style="color:var(--border-strong);">/</span><span style="color:var(--text-primary);font-weight:500;">dev-api</span></nav>',
    }));
    const actionSnippet = createRawSnippet(() => ({
      render: () =>
        '<button type="button" style="border:1px solid var(--border-default);background:var(--surface-default);color:var(--text-primary);border-radius:10px;padding:8px 16px;font-weight:600;font-size:var(--text-sm);">Promote</button>',
    }));

    const scenarios: Record<
      string,
      {
        title: string;
        description?: string;
        withEyebrow?: boolean;
        withActions?: boolean;
      }
    > = {
      "all slots": {
        description: "Cockpit for the payments platform.",
        title: "Overview",
        withActions: true,
        withEyebrow: true,
      },
      "title and description": {
        description:
          "Inspect the artifact deployments backing this project's vectors, stages, and landscapes.",
        title: "Artifact Deployments",
      },
      "title only": { title: "Landscapes" },
      "title with actions": { title: "Overview", withActions: true },
      "title with eyebrow": { title: "dev-api", withEyebrow: true },
    };

    for (const mode of ["light", "dark"]) {
      for (const [name, scenario] of Object.entries(scenarios)) {
        it(`renders ${mode} ${name}`, async () => {
          await page.viewport(720, 220);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.setAttribute("data-mode", mode);
          render(PageHeaderFixture, {
            actions: scenario.withActions ? actionSnippet : undefined,
            description: scenario.description,
            eyebrow: scenario.withEyebrow ? eyebrowSnippet : undefined,
            title: scenario.title,
          });
          await expect.element(page.getByTestId("page-header-root")).toMatchScreenshot();
        });
      }
    }
  });
});
