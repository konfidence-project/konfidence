import { describe, expect, it } from "vitest";

import { dashboardTitle } from "./dashboard";

describe("dashboard", () => {
  it("provides the application title", () => {
    expect(dashboardTitle).toBe("Konfidence Dashboard");
  });
});
