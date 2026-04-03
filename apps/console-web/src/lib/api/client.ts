export interface APIResponse<T = any> {
  request_id: string;
  data: T;
  error: { code: string; message: string } | null;
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

export type RequestOptions = {
  /** Abort the request after this many milliseconds (e.g. long-running LLM calls). */
  timeoutMs?: number;
};

async function request<T>(
  method: string,
  endpoint: string,
  body?: any,
  options?: RequestOptions
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const controller =
    options?.timeoutMs != null && options.timeoutMs > 0 ? new AbortController() : undefined;
  const timeoutId =
    controller != null ? setTimeout(() => controller.abort(), options!.timeoutMs) : undefined;

  let res: Response;
  try {
    res = await fetch(`/api/v1${endpoint}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
      signal: controller?.signal,
    });
  } finally {
    if (timeoutId != null) clearTimeout(timeoutId);
  }

  const json: APIResponse<T> = await res.json();

  if (!res.ok || json.error) {
    const code = json.error?.code || "unknown";
    const message = json.error?.message || res.statusText;

    if (res.status === 401) {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
        window.location.href = "/login";
      }
    }

    throw new APIError(code, message, res.status);
  }

  return json.data;
}

export class APIError extends Error {
  code: string;
  status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
    this.name = "APIError";
  }
}

export const api = {
  get: <T = any>(endpoint: string, options?: RequestOptions) => request<T>("GET", endpoint, undefined, options),
  post: <T = any>(endpoint: string, body?: any, options?: RequestOptions) =>
    request<T>("POST", endpoint, body, options),
  put: <T = any>(endpoint: string, body?: any, options?: RequestOptions) =>
    request<T>("PUT", endpoint, body, options),
  delete: <T = any>(endpoint: string, options?: RequestOptions) => request<T>("DELETE", endpoint, undefined, options),
};
