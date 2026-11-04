import { expect, test } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs, promotion } from "./fixtures.ts";
import { API, PATH } from "./helpers.ts";

test("renders real gate order, satisfied counts and pending approval without selecting the row", async ({
  page,
}) => {
  const gates = [
    { state: "Satisfied", type: "test.unit.example.com" },
    { state: "Satisfied", type: "test.e2e.example.com" },
    { state: "Error", type: "security.example.com" },
    { state: "Pending", type: "checks.example.com" },
    { state: "Pending", type: "explicit-approval.konfidence.cloud" },
  ];
  await page.route(API, (route) =>
    route.fulfill({
      json: { data: [{ ...configs[1], promotions: [{ ...promotion("gated", 1), gates }] }] },
    }),
  );
  await signIn(page, PATH, PATH);
  const trigger = page.getByRole("button", { name: "Promotion configuration a-flow" });
  await trigger.click();
  const strip = page.getByRole("group", { name: "2 of 5 gates satisfied" });
  await expect(strip).toBeVisible();
  await expect(strip.getByRole("img")).toHaveCount(5);
  await Promise.all(
    [
      "test.unit.example.com: Satisfied",
      "test.e2e.example.com: Satisfied",
      "security.example.com: Error",
      "checks.example.com: Pending",
      "Promotion approval: Pending",
    ].map((label, index) =>
      expect(strip.getByRole("img").nth(index)).toHaveAttribute("aria-label", label),
    ),
  );
  await strip.click();
  await expect(page.getByRole("button", { name: "View promotion gated" })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("button", { exact: true, name: "Approve" })).toBeDisabled();
});

test("missing gate information does not invent a successful zero-gate progress strip", async ({
  page,
}) => {
  await page.route(API, (route) =>
    route.fulfill({
      json: {
        data: [{ ...configs[0], promotions: [promotion("no-gates", 1, { gates: undefined })] }],
      },
    }),
  );
  await signIn(page, PATH, PATH);
  await page.getByRole("button", { name: "Promotion configuration z-template" }).click();
  await expect(page.getByRole("group", { name: /gates satisfied/ })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "View promotion no-gates" })).toBeVisible();
});
