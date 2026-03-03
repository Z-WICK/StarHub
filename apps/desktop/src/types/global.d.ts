type SessionTokenBridge = {
  set(token: string): Promise<void>;
  get(): Promise<string>;
  clear(): Promise<void>;
};

declare global {
  interface Window {
    sessionToken?: SessionTokenBridge;
  }
}

export {};
