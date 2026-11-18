import { expect, test } from "@playwright/test";

test("every catalog link has one mounted destination with unique IDs", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/");
  const nav = page.getByRole("navigation", { exact: true, name: "Components" });
  await expect(nav.getByRole("link", { exact: true, name: "Table" })).toBeAttached();
  const missing = await nav
    .getByRole("link")
    .evaluateAll((links) =>
      links
        .filter(
          (link) =>
            !document.getElementById(
              new URL(link.getAttribute("href")!, location.href).hash.slice(1),
            ),
        )
        .map((link) => link.textContent),
    );
  expect(missing).toEqual([]);
  const duplicates = await page.evaluate(() => {
    const ids = [...document.querySelectorAll("[id]")].map((element) => element.id);
    return ids.filter((id, index) => ids.indexOf(id) !== index);
  });
  expect(duplicates).toEqual([]);
  expect(errors).toEqual([]);
});
