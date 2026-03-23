export interface APIResponse<T = any> {
  request_id: string;
  data: T;
  error: { code: string; message: string } | null;
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

async function request<T>(
  method: string,
  endpoint: string,
  body?: any
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`/api/v1${endpoint}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

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
  get: <T = any>(endpoint: string) => request<T>("GET", endpoint),
  post: <T = any>(endpoint: string, body?: any) => request<T>("POST", endpoint, body),
  put: <T = any>(endpoint: string, body?: any) => request<T>("PUT", endpoint, body),
  delete: <T = any>(endpoint: string) => request<T>("DELETE", endpoint),
};
