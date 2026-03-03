import { contextBridge, ipcRenderer } from "electron";

export type SessionTokenBridge = {
  set(token: string): Promise<void>;
  get(): Promise<string>;
  clear(): Promise<void>;
};

const sessionToken: SessionTokenBridge = {
  set(token: string) {
    return ipcRenderer.invoke("session-token:set", token);
  },
  get() {
    return ipcRenderer.invoke("session-token:get");
  },
  clear() {
    return ipcRenderer.invoke("session-token:clear");
  },
};

contextBridge.exposeInMainWorld("sessionToken", sessionToken);
