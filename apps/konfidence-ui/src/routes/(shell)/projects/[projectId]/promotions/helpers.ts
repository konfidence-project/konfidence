import type { Page } from "@playwright/test";
import { signIn } from "../../../../../../e2e/helpers.ts";
import { configs } from "./fixtures.ts";

const PATH = "/projects/payments-platform/promotions";
const API = "**/v1/projects/payments-platform/vectorPromotionConfigs";
const openPromotions = async (page: Page, path = PATH): Promise<void> => {
  await page.route(API, (route) => route.fulfill({ json: { data: configs } }));
  await signIn(page, path, path);
};

export { API, openPromotions, PATH };
