import { expect, test } from "@playwright/test";

const luminance = (rgb: readonly number[]): number =>
  rgb
    .slice(0, 3)
    .map((value) => {
      const channel = value / 255;
      return channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4;
    })
    .reduce((sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index], 0);
const contrast = (left: readonly number[], right: readonly number[]): number => {
  const leftLuminance = luminance(left);
  const rightLuminance = luminance(right);
  return (
    (Math.max(leftLuminance, rightLuminance) + 0.05) /
    (Math.min(leftLuminance, rightLuminance) + 0.05)
  );
};

for (const colorScheme of ["light", "dark"] as const) {
  test(`code selection is distinct and readable in ${colorScheme} mode`, async ({ page }) => {
    await page.emulateMedia({ colorScheme });
    await page.goto("/#component-Button");
    const sample = page
      .locator("#component-Button")
      .getByRole("figure")
      .filter({
        has: page.getByRole("tablist", { exact: true, name: "Example tabs: Variants" }),
      });
    await sample.getByRole("tab", { exact: true, name: "Code" }).click();
    const editor = sample.getByRole("textbox");
    await editor.focus();
    await page.keyboard.press("Shift+End");
    await expect(sample.locator(".cm-selectionBackground").first()).toBeVisible();
    await expect(sample.locator(".cm-activeLine")).toHaveCSS(
      "background-color",
      "rgba(0, 0, 0, 0)",
    );
    const colors = await sample.locator(".cm-editor").evaluate((root) => {
      const canvas = document.createElement("canvas");
      canvas.width = 1;
      canvas.height = 1;
      const context = canvas.getContext("2d")!;
      const background = getComputedStyle(root).backgroundColor;
      const selection = getComputedStyle(
        root.querySelector(".cm-selectionBackground")!,
      ).backgroundColor;
      const token = root.querySelector(".cm-line span") ?? root.querySelector(".cm-content")!;
      const foreground = getComputedStyle(token, "::selection").color;
      context.fillStyle = background;
      context.fillRect(0, 0, 1, 1);
      const base = [...context.getImageData(0, 0, 1, 1).data];
      context.fillStyle = selection;
      context.fillRect(0, 0, 1, 1);
      const selected = [...context.getImageData(0, 0, 1, 1).data];
      context.clearRect(0, 0, 1, 1);
      context.fillStyle = foreground;
      context.fillRect(0, 0, 1, 1);
      return { base, foreground: [...context.getImageData(0, 0, 1, 1).data], selected };
    });
    expect(
      contrast(colors.base, colors.selected),
      "Selected code must be visibly different from its background",
    ).toBeGreaterThanOrEqual(1.5);
    expect(
      contrast(colors.foreground, colors.selected),
      "Selected code must remain readable",
    ).toBeGreaterThanOrEqual(4.5);
    await page.emulateMedia({ colorScheme: colorScheme === "dark" ? "light" : "dark" });
    await expect(page.locator("html")).toHaveClass(
      colorScheme === "dark" ? /^(?!.*\bdark\b)/ : /\bdark\b/,
    );
    await expect(sample.locator(".cm-editor")).toHaveClass(/cm-has-selection/);
    await expect(sample.locator(".cm-activeLine")).toHaveCSS(
      "background-color",
      "rgba(0, 0, 0, 0)",
    );
  });
}
