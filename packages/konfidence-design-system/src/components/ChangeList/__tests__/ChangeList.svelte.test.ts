import { page } from "vitest/browser";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";
import ChangeList from "../ChangeList.svelte";
import ChangeListFixture from "./ChangeListFixture.svelte";
import "../../../styles/test.css";

describe("ChangeList", () => {
  it("renders only applicable values for Added and Removed", async () => {
    await render(ChangeList, {
      items: [
        { id: "a", name: "added", newValue: "1.0", oldValue: "ignore-old", state: "added" },
        { id: "r", name: "removed", newValue: "ignore-new", oldValue: "0.9", state: "removed" },
      ],
    });
    await expect.element(page.getByRole("list", { name: "Changes" })).toBeInTheDocument();
    await expect.element(page.getByText("New value: 1.0")).toBeInTheDocument();
    await expect.element(page.getByText("Previous value: 0.9")).toBeInTheDocument();
    await expect.element(page.getByText("ignore-old")).not.toBeInTheDocument();
    await expect.element(page.getByText("ignore-new")).not.toBeInTheDocument();
  });

  it("preserves false and renders read-only toggle transitions", async () => {
    await render(ChangeList, {
      items: [
        {
          id: "e",
          label: "Enabled",
          name: "search",
          newValue: true,
          oldValue: false,
          state: "changed",
        },
        {
          id: "d",
          label: "Disabled",
          name: "checkout",
          newValue: false,
          oldValue: true,
          state: "changed",
        },
      ],
    });
    const rows = page.getByRole("listitem");
    await expect
      .element(rows.nth(0))
      .toHaveTextContent("Enabled search Previous value: Off → New value: On");
    await expect
      .element(rows.nth(1))
      .toHaveTextContent("Disabled checkout Previous value: On → New value: Off");
    await expect.element(page.getByRole("switch")).not.toBeInTheDocument();
  });

  it("omits the value area when values are absent and labels consecutive groups once", async () => {
    const screen = await render(ChangeList, {
      items: [
        { group: "Artifacts", id: "a", name: "first", state: "added" },
        { group: "Artifacts", id: "b", name: "second", state: "changed" },
        { group: "Toggles", id: "c", name: "third", state: "removed" },
      ],
    });
    expect(screen.container.querySelectorAll(".change-val")).toHaveLength(0);
    expect(screen.container.querySelectorAll(".change-group-label")).toHaveLength(2);
    await expect.element(page.getByText("Artifacts", { exact: true })).toBeInTheDocument();
  });

  it("replaces the list with the consumer's inline empty hint", async () => {
    await render(ChangeList, { emptyLabel: "No feature toggle changes.", items: [] });
    await expect.element(page.getByText("No feature toggle changes.")).toBeInTheDocument();
    await expect.element(page.getByRole("list")).not.toBeInTheDocument();
  });

  describe("reference screenshots", () => {
    afterEach(() => {
      document.documentElement.removeAttribute("data-theme");
      document.documentElement.classList.remove("dark");
    });
    for (const mode of ["light", "dark"]) {
      for (const width of [320, 520]) {
        it(`renders ${mode} at ${width}px without overflow`, async () => {
          await page.viewport(760, 800);
          document.documentElement.setAttribute("data-theme", "konfidence");
          document.documentElement.classList.toggle("dark", mode === "dark");
          const screen = await render(ChangeListFixture, { width });
          const fixture = screen.container.querySelector<HTMLElement>(".change-list-fixture")!;
          expect(fixture.scrollWidth).toBe(fixture.clientWidth);
          for (const row of fixture.querySelectorAll<HTMLElement>(".change-row")) {
            expect(row.scrollWidth).toBe(row.clientWidth);
          }
          await expect
            .element(page.getByText("international-payment-reconciliation-service"))
            .toHaveAttribute("title", "international-payment-reconciliation-service");
          await expect
            .element(
              page.getByRole("list", { name: "Example changes" }).element().parentElement!
                .parentElement!,
            )
            .toMatchScreenshot();
        });
      }
    }
  });
});
