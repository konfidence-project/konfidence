import { page } from "vitest/browser";
import { describe, expect, it } from "vitest";
import { render } from "vitest-browser-svelte";

import SelectFixture from "./SelectFixture.svelte";
import "../../../styles/test.css";

describe("<Select>", () => {
  it("renders every option passed as children", async () => {
    await render(SelectFixture);
    const combo = page.getByRole("combobox", { name: "Filter by status" });
    await expect.element(combo).toBeInTheDocument();
    await expect.element(page.getByRole("option", { name: "All statuses" })).toBeInTheDocument();
    await expect.element(page.getByRole("option", { name: "Deployed" })).toBeInTheDocument();
    await expect.element(page.getByRole("option", { name: "Fetched" })).toBeInTheDocument();
  });

  it("reflects the bound value", async () => {
    await render(SelectFixture, { value: "deployed" });
    const combo = page.getByRole("combobox", { name: "Filter by status" });
    await expect.element(combo).toHaveValue("deployed");
  });

  it("is disabled when the disabled prop is set", async () => {
    await render(SelectFixture, { disabled: true });
    const combo = page.getByRole("combobox", { name: "Filter by status" });
    await expect.element(combo).toBeDisabled();
  });
});
