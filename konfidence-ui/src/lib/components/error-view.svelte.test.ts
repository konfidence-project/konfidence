import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

const expectSingleRecoveryLink = (container: HTMLElement) => {
  const link = container.querySelector<HTMLAnchorElement>(".action-link");

  expect(link).not.toBeNull();
  expect(link?.textContent).toBe("Back to start");
  expect(link?.getAttribute("href")).toBe("/");
  expect(container.querySelector("button")).toBeNull();
  expect(container.querySelectorAll("a")).toHaveLength(1);
};

const getSupernovaParts = (container: HTMLElement) => ({
  aura: container.querySelector<HTMLElement>(".aura"),
  contentCard: container.querySelector<HTMLElement>(".content-card"),
  errorArt: container.querySelector<HTMLImageElement>(".error-art"),
  orbit: container.querySelector<HTMLElement>(".orbit-outer"),
  supernova: container.querySelector<HTMLElement>(".supernova"),
});

const expectSupernovaStructure = (container: HTMLElement) => {
  const { aura, contentCard, errorArt, orbit, supernova } = getSupernovaParts(container);
  expect(supernova).not.toBeNull();
  expect(errorArt?.tagName).toBe("IMG");
  expect(aura).not.toBeNull();
  expect(orbit).not.toBeNull();
  expect(contentCard).not.toBeNull();
};

const expectSupernovaLayout = (container: HTMLElement) => {
  const { contentCard, supernova } = getSupernovaParts(container);

  expect(supernova?.getBoundingClientRect().top).toBeLessThan(
    contentCard?.getBoundingClientRect().top ?? 0,
  );
};

const expectSupernovaAnimations = (container: HTMLElement) => {
  const { aura, errorArt, orbit } = getSupernovaParts(container);

  expect(globalThis.getComputedStyle(aura as Element).animationName).toContain("supernova-pulse");
  expect(globalThis.getComputedStyle(orbit as Element).animationName).toContain("orbit-drift");
  expect(globalThis.getComputedStyle(errorArt as Element).animationName).toContain("art-float");
};

const expectAnimatedSupernova = (container: HTMLElement) => {
  expectSupernovaStructure(container);
  expectSupernovaLayout(container);
  expectSupernovaAnimations(container);
};

describe("ErrorView", () => {
  it("renders the animated supernova error view with a single start recovery link", async () => {
    const { default: ErrorView } = await import("./ErrorView.svelte");
    const screen = await render(ErrorView, {
      props: {
        error: new Error("Loader returned 503"),
        message: "The dashboard could not render this route.",
        status: 500,
        title: "Something exploded",
      },
    });

    await expect.element(screen.getByText("Error 500")).toBeVisible();
    await expect.element(screen.getByText("Something exploded")).toBeVisible();
    await expect
      .element(screen.getByText("The dashboard could not render this route."))
      .toBeVisible();
    await expect.element(screen.getByText("Loader returned 503")).toBeVisible();

    expectSingleRecoveryLink(screen.container);
    expectAnimatedSupernova(screen.container);
  });
});
