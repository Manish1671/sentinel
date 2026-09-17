export class ApiError extends Error {
  readonly status: number;
  readonly code: string | null;
  readonly requestId: string | null;
  readonly body: unknown;

  constructor(options: {
    message: string;
    status: number;
    code?: string | null;
    requestId?: string | null;
    body?: unknown;
  }) {
    super(options.message);
    this.name = "ApiError";
    this.status = options.status;
    this.code = options.code ?? null;
    this.requestId = options.requestId ?? null;
    this.body = options.body ?? null;
  }
}

export type ApiErrorBody = {
  error?: {
    code?: string;
    message?: string;
    request_id?: string;
  };
};
