export type RequestState<T> =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "success"; data: T }
  | { status: "error"; error: Error };

export const idleState = { status: "idle" } as const;

export function loadingState(): RequestState<never> {
  return { status: "loading" };
}

export function successState<T>(data: T): RequestState<T> {
  return { status: "success", data };
}

export function errorState(error: Error): RequestState<never> {
  return { status: "error", error };
}
