import { NextResponse } from "next/server";

const API_URL = process.env.API_URL ?? process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";

export async function GET() {
  try {
    const upstream = await fetch(`${API_URL}/ready`, { cache: "no-store" });
    const body = await upstream.text();
    return new NextResponse(body || JSON.stringify({ status: upstream.ok ? "ok" : "unavailable" }), {
      status: upstream.status,
      headers: { "Content-Type": upstream.headers.get("content-type") ?? "application/json" },
    });
  } catch {
    return NextResponse.json(
      { status: "unavailable", checks: { control_plane: "unreachable" } },
      { status: 503 },
    );
  }
}
