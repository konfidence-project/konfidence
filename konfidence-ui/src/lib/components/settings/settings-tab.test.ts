import {
  DEFAULT_SETTINGS_TAB,
  SETTINGS_TAB_IDS,
  SETTINGS_URL_PARAM,
  parseSettingsTab,
} from "./settings-tab.js";
import { describe, expect, it } from "vitest";

describe("settings-tab", () => {
  it("has profile as the default tab", () => {
    expect(DEFAULT_SETTINGS_TAB).toBe("profile");
  });

  it("exposes profile, appearance, and landscape as the known ids", () => {
    expect(SETTINGS_TAB_IDS).toEqual(["profile", "appearance", "landscape"]);
  });

  it("uses `settings` as the URL param", () => {
    expect(SETTINGS_URL_PARAM).toBe("settings");
  });

  describe("parseSettingsTab", () => {
    it("returns undefined for missing input", () => {
      expect(parseSettingsTab(undefined)).toBeUndefined();
      expect(parseSettingsTab("")).toBeUndefined();
    });

    it("returns undefined for unknown tab ids", () => {
      expect(parseSettingsTab("bogus")).toBeUndefined();
      expect(parseSettingsTab("PROFILE")).toBeUndefined();
    });

    it("returns the tab id for each known value", () => {
      expect(parseSettingsTab("profile")).toBe("profile");
      expect(parseSettingsTab("appearance")).toBe("appearance");
      expect(parseSettingsTab("landscape")).toBe("landscape");
    });
  });
});
