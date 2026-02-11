/**
 * API client with automatic token refresh on 401 responses.
 *
 * The access token is stored in-memory only (never localStorage)
 * for security. On 401, we attempt a single silent refresh via
 * the httpOnly refresh cookie before failing.
 */

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
  accessToken = token;
}

export function getAccessToken(): string | null {
  return accessToken;
}

type ApiResponse<T> = {
  data: T;
};

type ApiError = {
  error: string;
  status: number;
};

export class ApiRequestError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
  }
}

async function request<T>(
  url: string,
  options: RequestInit = {},
  retry = true,
): Promise<T> {
  const headers = new Headers(options.headers);

  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  if (options.body && typeof options.body === "string" && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(url, { ...options, headers, credentials: "include" });

  if (response.status === 401 && retry) {
    const refreshed = await silentRefresh();
    if (refreshed) {
      return request<T>(url, options, false);
    }
  }

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`;
    try {
      const body = (await response.json()) as ApiError;
      if (body.error) {
        message = body.error;
      }
    } catch {
      // Response body wasn't JSON -- use the default message
    }
    throw new ApiRequestError(message, response.status);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as never;
  }

  const json = (await response.json()) as ApiResponse<T>;
  return json.data;
}

let refreshPromise: Promise<boolean> | null = null;

async function doSilentRefresh(): Promise<boolean> {
  try {
    const response = await fetch("/admin/api/auth/refresh", {
      method: "POST",
      credentials: "include",
    });

    if (!response.ok) {
      accessToken = null;
      return false;
    }

    const json = (await response.json()) as ApiResponse<{ access_token: string }>;
    accessToken = json.data.access_token;
    return true;
  } catch {
    accessToken = null;
    return false;
  }
}

function silentRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = doSilentRefresh().finally(() => {
    refreshPromise = null;
  });
  return refreshPromise;
}

export const api = {
  get<T>(url: string): Promise<T> {
    return request<T>(url);
  },

  post<T>(url: string, body?: unknown): Promise<T> {
    return request<T>(url, {
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    });
  },

  put<T>(url: string, body?: unknown): Promise<T> {
    return request<T>(url, {
      method: "PUT",
      body: body ? JSON.stringify(body) : undefined,
    });
  },

  patch<T>(url: string, body?: unknown): Promise<T> {
    return request<T>(url, {
      method: "PATCH",
      body: body ? JSON.stringify(body) : undefined,
    });
  },

  delete<T>(url: string): Promise<T> {
    return request<T>(url, { method: "DELETE" });
  },

  /** Attempt a silent refresh on page load to restore the session. */
  silentRefresh,
};
