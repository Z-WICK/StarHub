import { expect, test } from "@playwright/test";

const REAL_SESSION_TOKEN = "e2e_real_backend_session_token";

test("real backend session persists after reload", async ({ page }) => {
  await page.addInitScript((token) => {
    window.sessionStorage.setItem("gsm_session_token", token);
  }, REAL_SESSION_TOKEN);

  await page.goto("/#/stars");

  await expect(page).toHaveURL(/#\/stars/);
  await expect(page.getByRole("heading", { name: "Stars 列表" })).toBeVisible();

  await page.reload();

  await expect(page).toHaveURL(/#\/stars/);
  await expect(page.getByRole("heading", { name: "Stars 列表" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "GitHub 登录" })).toHaveCount(0);
});
