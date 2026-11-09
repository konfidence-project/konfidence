import { commands } from "vitest/browser";
import { beforeEach } from "vitest";

declare module "vitest/browser" {
  interface BrowserCommands {
    resetPointer: () => Promise<void>;
  }
}

beforeEach(async () => {
  // Vitest reuses browser pages, including the pointer position, between test files.
  // Reset before the test so explicit hover interactions still work normally.
  await commands.resetPointer();
});
