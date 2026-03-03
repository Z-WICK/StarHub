import { expect, test, _electron as electron } from "@playwright/test";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const desktopRoot = path.resolve(__dirname, "..");

test("electron app loads and shows login", async () => {
  const app = await electron.launch({
    args: ["."],
    cwd: desktopRoot,
    env: {
      ...process.env,
      ELECTRON_RENDERER_URL: "http://127.0.0.1:1420",
    },
  });

  const page = await app.firstWindow();
  await expect(
    page.getByRole("heading", { name: "GitHub 登录" }),
  ).toBeVisible();

  await app.close();
});
