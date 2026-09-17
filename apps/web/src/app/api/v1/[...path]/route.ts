import { NextRequest, NextResponse } from "next/server";
import { SESSION_COOKIE } from "@/lib/auth/cookie";

const API_URL = process.env.API_URL ?? process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";

type RouteContext = { params: Promise<{ path: string[] }> };

async function proxy(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  const targetPath = `/api/v1/${path.join("/")}`;
  const url = `${API_URL}${targetPath}${request.nextUrl.search}`;
  const headers = new Headers();
  const contentType = request.headers.get("content-type");
  if (contentType) headers.set("Content-Type", contentType);
  const idempotency = request.headers.get("idempotency-key");
  if (idempotency) headers.set("Idempotency-Key", idempotency);
  const requestId = request.headers.get("x-request-id");
  if (requestId) headers.set("X-Request-Id", requestId);

  const incomingAuth = request.headers.get("authorization");
  const session = request.cookies.get(SESSION_COOKIE)?.value;
  if (incomingAuth) {
    headers.set("Authorization", incomingAuth);
  } else if (session) {
    headers.set("Authorization", `Bearer ${session}`);
  }

  const method = request.method.toUpperCase();
  const body = method === "GET" || method === "HEAD" ? undefined : await request.arrayBuffer();

  let upstream: Response;
  try {
    upstream = await fetch(url, {
      method,
      headers,
      body,
      redirect: "manual",
      cache: "no-store",
    });
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "service_unavailable",
          message: "Unable to reach the Sentinel control plane.",
          details: {},
          request_id: requestId,
        },
      },
      { status: 503 },
    );
  }

  const isLogin = targetPath === "/api/v1/auth/login";
  const isLogout = targetPath === "/api/v1/auth/logout";
  const raw = await upstream.arrayBuffer();
  const responseHeaders = new Headers();
  const upstreamType = upstream.headers.get("content-type");
  if (upstreamType) responseHeaders.set("Content-Type", upstreamType);
  const upstreamRequestId = upstream.headers.get("x-request-id");
  if (upstreamRequestId) responseHeaders.set("X-Request-Id", upstreamRequestId);

  if (isLogin && upstream.ok) {
    try {
      const json = JSON.parse(new TextDecoder().decode(raw)) as {
        data?: { token?: string; user?: unknown };
      };
      const token = json.data?.token;
      if (token && json.data) {
        delete json.data.token;
        const loginResponse = new NextResponse(JSON.stringify(json), {
          status: upstream.status,
          headers: { "Content-Type": "application/json" },
        });
        loginResponse.cookies.set({
          name: SESSION_COOKIE,
          value: token,
          httpOnly: true,
          sameSite: "lax",
          secure: process.env.NODE_ENV === "production",
          path: "/",
          maxAge: 60 * 60 * 12,
        });
        if (upstreamRequestId) loginResponse.headers.set("X-Request-Id", upstreamRequestId);
        return loginResponse;
      }
    } catch {
      // keep original body
    }
  }

  const status = upstream.status;
  const allowBody = status !== 204 && status !== 205 && status !== 304;
  const response = new NextResponse(allowBody ? raw : null, {
    status,
    headers: allowBody ? responseHeaders : undefined,
  });

  if (isLogout) {
    response.cookies.set({
      name: SESSION_COOKIE,
      value: "",
      httpOnly: true,
      sameSite: "lax",
      secure: process.env.NODE_ENV === "production",
      path: "/",
      maxAge: 0,
    });
  }

  return response;
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
