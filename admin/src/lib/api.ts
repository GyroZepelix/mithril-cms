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

type ApiResponseWithMeta<T, M> = {
  data: T;
  meta: M;
};

type ApiErrorBody = {
  error: string | { code: string; message: string; details?: unknown[] };
  status: number;
};

export class ApiRequestError extends Error {
  status: number;
  body: unknown;

  constructor(message: string, status: number, body?: unknown) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.body = body;
  }
}

async function parseErrorBody(response: Response): Promise<{ message: string; raw: unknown }> {
  try {
    const body = (await response.json()) as ApiErrorBody;
    if (typeof body.error === "string") {
      return { message: body.error, raw: body };
    }
    if (body.error && typeof body.error === "object") {
      return { message: body.error.message, raw: body };
    }
  } catch {
    // Response body wasn't JSON
  }
  return { message: `Request failed with status ${response.status}`, raw: undefined };
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
    const errorBody = await parseErrorBody(response);
    throw new ApiRequestError(errorBody.message, response.status, errorBody.raw);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as never;
  }

  const json = (await response.json()) as ApiResponse<T>;
  return json.data;
}

/**
 * Like request(), but returns {data, meta} instead of unwrapping data.
 * Used for paginated endpoints that return metadata alongside the data array.
 */
async function requestWithMeta<T, M>(
  url: string,
  options: RequestInit = {},
  retry = true,
): Promise<{ data: T; meta: M }> {
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
      return requestWithMeta<T, M>(url, options, false);
    }
  }

  if (!response.ok) {
    const errorBody = await parseErrorBody(response);
    throw new ApiRequestError(errorBody.message, response.status, errorBody.raw);
  }

  const json = (await response.json()) as ApiResponseWithMeta<T, M>;
  return { data: json.data, meta: json.meta };
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

  getWithMeta<T, M>(url: string): Promise<{ data: T; meta: M }> {
    return requestWithMeta<T, M>(url);
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
