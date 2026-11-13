import { expect, test } from "@playwright/test";

test("change list preview exposes all states, boolean transitions and empty sections", async ({
  page,
}) => {
  await page.goto("/#change-list");
  const section = page.locator("#change-list");
  await expect(section.getByRole("heading", { exact: true, name: "Change list" })).toBeVisible();
  const artifacts = section.getByRole("list", { exact: true, name: "Artifact changes" });
  await expect(artifacts.getByRole("listitem")).toHaveCount(4);
  await Promise.all(
    ["added", "changed", "removed"].map((state) =>
      expect(artifacts.locator(`[data-state="${state}"]`).first()).toBeVisible(),
    ),
  );
  const toggles = section.getByRole("list", { exact: true, name: "Feature toggle changes" });
  await expect(toggles.getByText("Enabled", { exact: true })).toBeVisible();
  await expect(toggles.getByText("Disabled", { exact: true })).toBeVisible();
  await expect(toggles.getByText("On", { exact: true })).toHaveCount(2);
  await expect(toggles.getByText("Off", { exact: true })).toHaveCount(2);
  await expect(
    section.getByText("No artifact changes in this promotion.", { exact: true }),
  ).toBeVisible();
  await expect(
    section.getByText("No feature toggle changes in this promotion.", { exact: true }),
  ).toBeVisible();
});
