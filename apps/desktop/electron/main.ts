import { app, BrowserWindow } from "electron";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { registerSessionTokenIpc } from "./ipc/sessionToken.js";

const TRUSTED_DEV_URLS = new Set([
  "http://127.0.0.1:1420",
  "http://localhost:1420",
]);

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packagedIndexFilePath = path.join(__dirname, "../dist/index.html");
const trustedFileUrlPrefix = `${pathToFileURL(path.dirname(packagedIndexFilePath)).href}`;

const resolveRendererUrl = (): string | null => {
  const rendererUrl = process.env.ELECTRON_RENDERER_URL;
  if (!rendererUrl) {
    return null;
  }

  if (app.isPackaged) {
    return null;
  }

  if (TRUSTED_DEV_URLS.has(rendererUrl)) {
    return rendererUrl;
  }

  return null;
};

const createMainWindow = async (): Promise<void> => {
  const mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    minWidth: 1024,
    minHeight: 640,
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  const rendererUrl = resolveRendererUrl();
  if (rendererUrl) {
    await mainWindow.loadURL(rendererUrl);
    return;
  }

  if (!app.isPackaged) {
    await mainWindow.loadURL("http://127.0.0.1:1420");
    return;
  }

  await mainWindow.loadFile(packagedIndexFilePath);
};

app.whenReady().then(async () => {
  registerSessionTokenIpc({ trustedFileUrlPrefix });
  await createMainWindow();

  app.on("activate", async () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      await createMainWindow();
    }
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});
