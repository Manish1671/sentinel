"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/lib/api/errors";

export type ResourceState<T> =
  | { status: "loading" }
  | { status: "success"; data: T }
  | { status: "error"; error: ApiError | Error };

export function useApiResource<T>(
  loader: (signal: AbortSignal) => Promise<T>,
  options: {
    enabled?: boolean;
    refreshMs?: number | ((data: T) => number | null);
    deps?: unknown[];
  } = {},
) {
  const { enabled = true, refreshMs, deps = [] } = options;
  const [state, setState] = useState<ResourceState<T>>({ status: "loading" });
  const loaderRef = useRef(loader);
  loaderRef.current = loader;
  const dataRef = useRef<T | null>(null);

  const load = useCallback(async (silent = false) => {
    if (!enabled) return;
    const controller = new AbortController();
    if (!silent) {
      setState((current) => (current.status === "success" ? current : { status: "loading" }));
    }
    try {
      const data = await loaderRef.current(controller.signal);
      dataRef.current = data;
      setState({ status: "success", data });
    } catch (error) {
      if (controller.signal.aborted) return;
      const err = error instanceof Error ? error : new Error("Request failed");
      if (!silent || dataRef.current === null) {
        setState({ status: "error", error: err });
      }
    }
    return controller;
  }, [enabled]);

  useEffect(() => {
    if (!enabled) return;
    const controller = new AbortController();
    let cancelled = false;
    setState({ status: "loading" });
    loaderRef.current(controller.signal)
      .then((data) => {
        if (cancelled) return;
        dataRef.current = data;
        setState({ status: "success", data });
      })
      .catch((error) => {
        if (cancelled || controller.signal.aborted) return;
        setState({
          status: "error",
          error: error instanceof Error ? error : new Error("Request failed"),
        });
      });
    return () => {
      cancelled = true;
      controller.abort();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, ...deps]);

  useEffect(() => {
    if (!enabled || refreshMs == null) return;
    const id = window.setInterval(() => {
      void load(true);
    }, typeof refreshMs === "function" ? (dataRef.current ? refreshMs(dataRef.current) ?? 20_000 : 20_000) : refreshMs);
    return () => window.clearInterval(id);
  }, [enabled, load, refreshMs, state.status]);

  return { ...state, reload: () => void load(false) };
}
