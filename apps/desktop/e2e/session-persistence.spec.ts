import { expect, test } from "@playwright/test";

type Envelope<T> = {
  success: boolean;
  data: T;
  error: null;
  meta: null | {
    page: number;
    limit: number;
    total: number;
  };
};

const ok = <T>(data: T, meta: Envelope<T>["meta"] = null): Envelope<T> => ({
  success: true,
  data,
  error: null,
  meta,
});

test("login persists after reload", async ({ page }, testInfo) => {
  await page.route("**/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const key = `${request.method()} ${url.pathname}`;

    if (key === "POST /v1/auth/login") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            token: "session_token_for_e2e",
            profile: {
              id: 1,
              displayName: "E2E User",
              githubLogin: "e2e-user",
              avatarUrl: "",
            },
          }),
        ),
      });
      return;
    }

    if (key === "GET /v1/auth/session") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            userId: 1,
          }),
        ),
      });
      return;
    }

    if (key === "GET /v1/stars") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok(
            {
              items: [
                {
                  repositoryId: 1001,
                  fullName: "demo/repo",
                  description: "demo",
                  language: "TypeScript",
                  stargazersCount: 1,
                  starredAt: new Date().toISOString(),
                  updatedAt: new Date().toISOString(),
                  pushedAt: new Date().toISOString(),
                  htmlUrl: "https://github.com/demo/repo",
                  note: "",
                  tags: [],
                },
              ],
            },
            { page: 1, limit: 20, total: 1 },
          ),
        ),
      });
      return;
    }

    if (key === "GET /v1/tags") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(ok([])),
      });
      return;
    }

    if (key === "GET /v1/sync/status") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            id: 1,
            status: "success",
            startedAt: new Date().toISOString(),
            finishedAt: new Date().toISOString(),
            cursor: "",
            errorMessage: "",
          }),
        ),
      });
      return;
    }

    await route.fulfill({
      status: 404,
      contentType: "application/json",
      body: JSON.stringify({
        success: false,
        data: null,
        error: `unmocked: ${key}`,
        meta: null,
      }),
    });
  });

  await page.goto("/");
  await page.getByPlaceholder("ghp_xxx").fill("ghp_test_token");
  await page.getByRole("button", { name: "登录" }).click();

  await expect(page).toHaveURL(/#\/stars/);
  await expect(page.getByRole("heading", { name: "Stars 列表" })).toBeVisible();

  await page.reload();

  await expect(page).toHaveURL(/#\/stars/);
  await expect(page.getByRole("heading", { name: "Stars 列表" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "GitHub 登录" })).toHaveCount(0);

  await page.screenshot({ path: testInfo.outputPath("logged-in-after-reload.png"), fullPage: true });
});
