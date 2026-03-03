import { app, ipcMain } from "electron";

type SessionTokenIpcOptions = {
  trustedFileUrlPrefix: string;
};

const TRUSTED_DEV_ORIGINS = new Set([
  "http://127.0.0.1:1420",
  "http://localhost:1420",
]);

type KeytarModule = {
  setPassword(
    service: string,
    account: string,
    password: string,
  ): Promise<void>;
  getPassword(service: string, account: string): Promise<string | null>;
  deletePassword(service: string, account: string): Promise<boolean>;
};

const SERVICE = "github-star-manager";
const ACCOUNT = "session-token";

let inMemoryToken = "";

const isDev = (): boolean => {
  return !app.isPackaged;
};

const loadKeytar = async (): Promise<KeytarModule | null> => {
  try {
    const imported = (await import("keytar")) as {
      default?: KeytarModule;
    } & KeytarModule;
    return imported.default ?? imported;
  } catch {
    return null;
  }
};

const setToken = async (token: string): Promise<void> => {
  const keytar = await loadKeytar();
  if (keytar) {
    await keytar.setPassword(SERVICE, ACCOUNT, token);
    return;
  }

  if (isDev()) {
    inMemoryToken = token;
    return;
  }

  throw new Error("Secure credential storage is unavailable");
};

const getToken = async (): Promise<string> => {
  if (inMemoryToken) {
    return inMemoryToken;
  }

  const keytar = await loadKeytar();
  if (keytar) {
    const token = await keytar.getPassword(SERVICE, ACCOUNT);
    return token ?? "";
  }

  if (isDev()) {
    return inMemoryToken;
  }

  throw new Error("Secure credential storage is unavailable");
};

const clearToken = async (): Promise<void> => {
  inMemoryToken = "";

  const keytar = await loadKeytar();
  if (keytar) {
    await keytar.deletePassword(SERVICE, ACCOUNT);
    return;
  }

  if (isDev()) {
    return;
  }

  throw new Error("Secure credential storage is unavailable");
};

const assertTrustedSender = (
  senderUrl: string,
  options: SessionTokenIpcOptions,
): void => {
  if (!senderUrl) {
    throw new Error("Untrusted IPC caller");
  }

  if (TRUSTED_DEV_ORIGINS.has(senderUrl)) {
    return;
  }

  if (senderUrl.startsWith(options.trustedFileUrlPrefix)) {
    return;
  }

  throw new Error("Untrusted IPC caller");
};

export const registerSessionTokenIpc = (
  options: SessionTokenIpcOptions,
): void => {
  ipcMain.handle("session-token:set", async (event, token: string) => {
    assertTrustedSender(event.senderFrame?.url ?? "", options);
    await setToken(token);
  });

  ipcMain.handle("session-token:get", async (event) => {
    assertTrustedSender(event.senderFrame?.url ?? "", options);
    return getToken();
  });

  ipcMain.handle("session-token:clear", async (event) => {
    assertTrustedSender(event.senderFrame?.url ?? "", options);
    await clearToken();
  });
};
