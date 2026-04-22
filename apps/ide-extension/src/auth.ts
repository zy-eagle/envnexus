import * as os from "os";
import * as vscode from "vscode";

export const API_BASE = "http://localhost:8080/api/v1";
/** Web console base URL where the user confirms device auth (`/device-auth/confirm?code=…`). */
const CONSOLE_DEVICE_AUTH_ORIGIN = "http://localhost:3000";

const SECRET_ACCESS = "access_token";
const SECRET_REFRESH = "refresh_token";
const SECRET_ACCESS_EXPIRES = "access_token_expires_at";

type ApiSuccess<T> = { data: T; error: null; request_id?: string };
type ApiErrorBody = { data: null; error: { code: string; message: string }; request_id?: string };

function isApiError(res: ApiSuccess<unknown> | ApiErrorBody): res is ApiErrorBody {
  return res.error != null;
}

interface DeviceInitData {
  device_code: string;
  user_code: string;
  expires_in: number;
  interval: number;
  verification_uri_complete?: string;
}

interface DevicePollData {
  error?: string;
  error_description?: string;
  access_token?: string;
  refresh_token?: string;
  expires_in?: number;
  token_type?: string;
  ide_client_token_id?: string;
}

interface RefreshData {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
  token_id: string;
}

let loginSeq = 0;
let loginPollHandle: ReturnType<typeof setInterval> | undefined;

async function readSecret(context: vscode.ExtensionContext, key: string): Promise<string | undefined> {
  const v = await context.secrets.get(key);
  return v ?? undefined;
}

function base64UrlDecode(s: string): string {
  const pad = 4 - (s.length % 4);
  const b64 = s.replace(/-/g, "+").replace(/_/g, "/") + (pad < 4 ? "=".repeat(pad) : "");
  return Buffer.from(b64, "base64").toString("utf8");
}

/** Returns true if the JWT (three segments) is expired, using `exp` when present. */
function isJwtExpired(accessToken: string, skewSec: number): boolean {
  const parts = accessToken.split(".");
  if (parts.length !== 3) {
    return false;
  }
  try {
    const payload = JSON.parse(base64UrlDecode(parts[1])) as { exp?: number };
    if (typeof payload.exp !== "number") {
      return false;
    }
    return Date.now() / 1000 >= payload.exp - skewSec;
  } catch {
    return false;
  }
}

function parseExpiresAtMs(stored: string | undefined): number | undefined {
  if (!stored) {
    return undefined;
  }
  const n = Number(stored);
  if (Number.isFinite(n) && n > 0) {
    return n;
  }
  return undefined;
}

async function saveTokens(
  context: vscode.ExtensionContext,
  access: string,
  refresh: string,
  expiresInSec: number
): Promise<void> {
  const expMs = Date.now() + Math.max(0, expiresInSec) * 1000;
  await context.secrets.store(SECRET_ACCESS, access);
  await context.secrets.store(SECRET_REFRESH, refresh);
  await context.secrets.store(SECRET_ACCESS_EXPIRES, String(expMs));
}

/** Stop device-auth polling (e.g. on extension deactivate or before a new login). */
export function disposeDeviceAuth(): void {
  if (loginPollHandle !== undefined) {
    clearInterval(loginPollHandle);
    loginPollHandle = undefined;
  }
  loginSeq++;
}

export async function login(context: vscode.ExtensionContext): Promise<void> {
  disposeDeviceAuth();
  const mySeq = loginSeq;
  const deviceInfo = JSON.stringify({ name: "VSCode", os: os.platform() });
  let init: ApiSuccess<DeviceInitData> | ApiErrorBody;
  try {
    init = await postJson<ApiSuccess<DeviceInitData> | ApiErrorBody>(`${API_BASE}/device-auth/init`, {
      device_info: deviceInfo
    });
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    void vscode.window.showErrorMessage(`EnvNexus login failed: ${msg}`);
    return;
  }
  if (isApiError(init)) {
    void vscode.window.showErrorMessage(`EnvNexus login failed: ${init.error.message}`);
    return;
  }
  const d = init.data;
  const intervalSec = Math.max(1, d.interval);
  const verificationUri =
    d.verification_uri_complete ??
    `${CONSOLE_DEVICE_AUTH_ORIGIN}/device-auth/confirm?code=${encodeURIComponent(d.user_code)}`;

  const choice = await vscode.window.showInformationMessage(
    `Please authenticate in your browser. User Code: ${d.user_code}`,
    "Open Browser"
  );
  if (mySeq !== loginSeq) {
    return;
  }
  if (choice === "Open Browser") {
    await vscode.env.openExternal(vscode.Uri.parse(verificationUri));
  }

  const runPoll = async (): Promise<boolean> => {
    if (mySeq !== loginSeq) {
      return true;
    }
    let res: ApiSuccess<DevicePollData> | ApiErrorBody;
    try {
      res = await postJson<ApiSuccess<DevicePollData> | ApiErrorBody>(`${API_BASE}/device-auth/poll`, {
        device_code: d.device_code
      });
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      void vscode.window.showErrorMessage(`EnvNexus login failed: ${msg}`);
      if (loginPollHandle !== undefined) {
        clearInterval(loginPollHandle);
        loginPollHandle = undefined;
      }
      return true;
    }
    if (isApiError(res)) {
      void vscode.window.showErrorMessage(`EnvNexus login failed: ${res.error.message}`);
      if (loginPollHandle !== undefined) {
        clearInterval(loginPollHandle);
        loginPollHandle = undefined;
      }
      return true;
    }
    const p = res.data;
    if (p.access_token && p.refresh_token) {
      if (mySeq !== loginSeq) {
        return true;
      }
      const expIn = p.expires_in ?? 3600;
      await saveTokens(context, p.access_token, p.refresh_token, expIn);
      void vscode.window.showInformationMessage("EnvNexus: signed in successfully.");
      return true;
    }
    const err = p.error ?? "";
    if (err === "authorization_pending" || err === "slow_down") {
      return false;
    }
    if (err === "access_denied") {
      void vscode.window.showErrorMessage(
        `EnvNexus: ${p.error_description ?? "The user denied the authorization request."}`
      );
      return true;
    }
    if (err === "expired_token") {
      void vscode.window.showErrorMessage(
        `EnvNexus: ${p.error_description ?? "The device code has expired. Start login again."}`
      );
      return true;
    }
    void vscode.window.showErrorMessage(
      p.error_description ?? `EnvNexus: unexpected poll response (error: ${err || "none"})`
    );
    return true;
  };

  const tick = async (): Promise<void> => {
    if (mySeq !== loginSeq) {
      return;
    }
    const done = await runPoll();
    if (done && loginPollHandle !== undefined) {
      clearInterval(loginPollHandle);
      loginPollHandle = undefined;
    }
  };

  loginPollHandle = setInterval(() => {
    void tick();
  }, intervalSec * 1000);
  await tick();
}

const ACCESS_SKEW_SEC = 30;

export async function getValidAccessToken(context: vscode.ExtensionContext): Promise<string | undefined> {
  const access = await readSecret(context, SECRET_ACCESS);
  const refresh = await readSecret(context, SECRET_REFRESH);
  const expStored = await readSecret(context, SECRET_ACCESS_EXPIRES);

  if (!access && !refresh) {
    return undefined;
  }

  const isExpired = (): boolean => {
    if (isJwtExpired(access ?? "", ACCESS_SKEW_SEC)) {
      return true;
    }
    const expMs = parseExpiresAtMs(expStored);
    if (expMs === undefined) {
      return false;
    }
    return Date.now() >= expMs - ACCESS_SKEW_SEC * 1000;
  };

  if (access && !isExpired()) {
    return access;
  }

  if (!refresh) {
    return undefined;
  }

  let res: ApiSuccess<RefreshData> | ApiErrorBody;
  try {
    res = await postJson<ApiSuccess<RefreshData> | ApiErrorBody>(`${API_BASE}/device-auth/refresh`, {
      refresh_token: refresh
    });
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    void vscode.window.showErrorMessage(`EnvNexus: could not refresh session — ${msg}`);
    return undefined;
  }
  if (isApiError(res)) {
    void vscode.window.showErrorMessage(`EnvNexus: could not refresh session — ${res.error.message}`);
    return undefined;
  }
  const r = res.data;
  await saveTokens(context, r.access_token, r.refresh_token, r.expires_in);
  return r.access_token;
}

async function postJson<T extends ApiErrorBody | ApiSuccess<unknown>>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(body)
  });
  const text = await res.text();
  let parsed: T;
  try {
    parsed = JSON.parse(text) as T;
  } catch {
    throw new Error(`Invalid JSON from ${url} (${res.status}): ${text.slice(0, 200)}`);
  }
  if (isApiError(parsed as ApiErrorBody | ApiSuccess<unknown>)) {
    return parsed;
  }
  if (!res.ok) {
    throw new Error(`Request to ${url} failed (${res.status})`);
  }
  return parsed;
}
