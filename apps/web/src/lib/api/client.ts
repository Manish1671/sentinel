import { ApiError, type ApiErrorBody } from "@/lib/api/errors";
import { SESSION_COOKIE } from "@/lib/auth/cookie";

export { SESSION_COOKIE };

export function getApiBaseUrl(): string {
  if (typeof window !== "undefined") {
    return "";
  }
  return process.env.API_URL ?? process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
}

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  idempotencyKey?: string;
  token?: string;
  timeoutMs?: number;
};

async function parseBody(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

function newRequestId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `req-${Date.now()}`;
}

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, idempotencyKey, token, headers, timeoutMs = 25000, signal, ...rest } = options;
  const requestHeaders = new Headers(headers);
  const requestId = requestHeaders.get("X-Request-Id") ?? newRequestId();
  requestHeaders.set("X-Request-Id", requestId);

  if (body !== undefined && !requestHeaders.has("Content-Type")) {
    requestHeaders.set("Content-Type", "application/json");
  }
  if (token) {
    requestHeaders.set("Authorization", `Bearer ${token}`);
  }
  if (idempotencyKey) {
    requestHeaders.set("Idempotency-Key", idempotencyKey);
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  const onAbort = () => controller.abort();
  if (signal) {
    if (signal.aborted) controller.abort();
    else signal.addEventListener("abort", onAbort, { once: true });
  }

  try {
    const response = await fetch(`${getApiBaseUrl()}${path}`, {
      ...rest,
      credentials: "include",
      headers: requestHeaders,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: controller.signal,
    });

    const parsed = await parseBody(response);
    const responseRequestId = response.headers.get("X-Request-Id") ?? requestId;

    if (!response.ok) {
      const errorBody = parsed as ApiErrorBody;
      throw new ApiError({
        message: errorBody.error?.message ?? `Request failed (${response.status})`,
        status: response.status,
        code: errorBody.error?.code ?? null,
        requestId: errorBody.error?.request_id ?? responseRequestId,
        body: parsed,
      });
    }

    return parsed as T;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if (error instanceof DOMException && error.name === "AbortError") {
      throw new ApiError({
        message: "Request timed out.",
        status: 408,
        code: "timeout",
      });
    }
    throw new ApiError({
      message: error instanceof Error ? error.message : "Network request failed.",
      status: 0,
      code: "network_error",
    });
  } finally {
    clearTimeout(timeout);
    if (signal) signal.removeEventListener("abort", onAbort);
  }
}

export function withQuery(path: string, query: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === "") continue;
    params.set(key, String(value));
  }
  const encoded = params.toString();
  return encoded ? `${path}?${encoded}` : path;
}
