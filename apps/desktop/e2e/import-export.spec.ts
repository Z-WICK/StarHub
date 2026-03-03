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

test("settings import/export flow works", async ({ page }) => {
  let lastImportBody: unknown = null;

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

    if (key === "GET /v1/stars") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({ items: [] }, { page: 1, limit: 20, total: 0 }),
        ),
      });
      return;
    }

    if (key === "GET /v1/tags") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok([
            { id: 11, name: "Go", color: "#2563eb" },
            { id: 12, name: "AI", color: "#7c3aed" },
          ]),
        ),
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

    if (key === "GET /v1/sync/settings") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            enabled: true,
            intervalHours: 12,
            retryMax: 2,
            updatedAt: new Date().toISOString(),
          }),
        ),
      });
      return;
    }

    if (key === "GET /v1/sync/rules") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(ok([])),
      });
      return;
    }

    if (key === "GET /v1/governance/metrics") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            totalStars: 10,
            untaggedStars: 2,
            untaggedRatio: 0.2,
            syncJobs7d: 4,
            syncSuccess7d: 3,
            syncSuccessRate7d: 0.75,
            staleStars: 1,
          }),
        ),
      });
      return;
    }

    if (key === "GET /v1/io/export") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            version: "v1",
            exportedAt: new Date().toISOString(),
            syncSettings: {
              enabled: true,
              intervalHours: 12,
              retryMax: 2,
              updatedAt: new Date().toISOString(),
            },
            tags: [
              { id: 11, name: "Go", color: "#2563eb" },
              { id: 12, name: "AI", color: "#7c3aed" },
            ],
            smartRules: [
              {
                name: "Go rules",
                enabled: true,
                languageEquals: "Go",
                ownerContains: "",
                nameContains: "",
                descriptionContains: "",
                tagName: "Go",
              },
            ],
            notes: [
              {
                githubRepoId: 123,
                content: "important note",
              },
            ],
            tagBindings: [
              {
                githubRepoId: 123,
                tagName: "Go",
              },
            ],
          }),
        ),
      });
      return;
    }

    if (key === "POST /v1/io/import") {
      lastImportBody = request.postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            tagsUpserted: 2,
            rulesUpserted: 1,
            notesUpserted: 1,
            tagBindingsLinked: 1,
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

  await page.getByRole("link", { name: "设置" }).click();
  await expect(page).toHaveURL(/#\/settings/);
  await expect(
    page.getByRole("heading", { name: "JSON 导入导出" }),
  ).toBeVisible();

  const textarea = page.getByPlaceholder("将导出的 JSON 粘贴到这里");
  await page.getByRole("button", { name: "导出配置" }).click();
  await expect(textarea).toHaveValue(/"version":\s*"v1"/);
  await expect(page.getByText("已生成导出 JSON，可复制保存。")).toBeVisible();

  await page.getByRole("button", { name: "导入配置" }).click();
  await expect(
    page.getByText("导入完成：标签 2、规则 1、备注 1、绑定 1"),
  ).toBeVisible();

  expect(lastImportBody).toBeTruthy();
  expect(lastImportBody).toMatchObject({
    tags: [{ name: "Go" }, { name: "AI" }],
    smartRules: [{ name: "Go rules" }],
    notes: [{ githubRepoId: 123 }],
    tagBindings: [{ tagName: "Go" }],
  });
});

test("settings import shows invalid JSON error", async ({ page }) => {
  let importCalled = false;

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

    if (key === "GET /v1/stars") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({ items: [] }, { page: 1, limit: 20, total: 0 }),
        ),
      });
      return;
    }

    if (key === "GET /v1/tags") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(ok([{ id: 11, name: "Go", color: "#2563eb" }])),
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

    if (key === "GET /v1/sync/settings") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            enabled: true,
            intervalHours: 12,
            retryMax: 2,
            updatedAt: new Date().toISOString(),
          }),
        ),
      });
      return;
    }

    if (key === "GET /v1/sync/rules") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(ok([])),
      });
      return;
    }

    if (key === "GET /v1/governance/metrics") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            totalStars: 1,
            untaggedStars: 1,
            untaggedRatio: 1,
            syncJobs7d: 1,
            syncSuccess7d: 1,
            syncSuccessRate7d: 1,
            staleStars: 0,
          }),
        ),
      });
      return;
    }

    if (key === "POST /v1/io/import") {
      importCalled = true;
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(
          ok({
            tagsUpserted: 0,
            rulesUpserted: 0,
            notesUpserted: 0,
            tagBindingsLinked: 0,
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

  await page.getByRole("link", { name: "设置" }).click();
  await expect(page).toHaveURL(/#\/settings/);

  const textarea = page.getByPlaceholder("将导出的 JSON 粘贴到这里");
  await textarea.fill("{ invalid json }");
  await page.getByRole("button", { name: "导入配置" }).click();

  await expect(page.getByText("JSON 格式无效")).toBeVisible();
  expect(importCalled).toBe(false);
});
