let inMemoryToken = "";
const WEB_TOKEN_KEY = "gsm_session_token";

const hasElectronBridge = (): boolean => {
  return (
    typeof window !== "undefined" && typeof window.sessionToken !== "undefined"
  );
};

export const saveSessionToken = async (token: string): Promise<void> => {
  if (hasElectronBridge()) {
    await window.sessionToken!.set(token);
    inMemoryToken = token;
    return;
  }

  inMemoryToken = token;
  if (typeof window !== "undefined") {
    window.sessionStorage.setItem(WEB_TOKEN_KEY, token);
  }
};

export const getSessionToken = async (): Promise<string> => {
  if (inMemoryToken) {
    return inMemoryToken;
  }

  if (hasElectronBridge()) {
    const token = await window.sessionToken!.get();
    inMemoryToken = token;
    return token;
  }

  if (typeof window !== "undefined") {
    const token = window.sessionStorage.getItem(WEB_TOKEN_KEY) ?? "";
    inMemoryToken = token;
    return token;
  }

  return "";
};

export const clearSessionToken = async (): Promise<void> => {
  inMemoryToken = "";

  if (hasElectronBridge()) {
    await window.sessionToken!.clear();
    return;
  }

  if (typeof window !== "undefined") {
    window.sessionStorage.removeItem(WEB_TOKEN_KEY);
  }
};
