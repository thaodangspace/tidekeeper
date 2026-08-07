/**
 * Domain errors with HTTP mapping.
 *
 * Mirrors the Go server's error taxonomy: domain errors carry a user-facing
 * code and, where relevant, a field path used for validation responses.
 */

export type ErrorKind =
  | "invalid_request"
  | "validation"
  | "unauthorized"
  | "not_found"
  | "conflict"
  | "rate_limited"
  | "service_unavailable"
  | "internal";

export interface ErrorOptions {
  code?: string;
  field?: string;
  cause?: unknown;
  retryable?: boolean;
}

export class TidekeepersError extends Error {
  readonly kind: ErrorKind;
  readonly code: string;
  readonly field?: string;
  readonly retryable: boolean;

  constructor(message: string, kind: ErrorKind, options: ErrorOptions = {}) {
    super(message);
    this.name = "TidekeepersError";
    this.kind = kind;
    this.code = options.code ?? defaultCode(kind);
    this.field = options.field;
    this.retryable = options.retryable ?? false;
  }
}

function defaultCode(kind: ErrorKind): string {
  switch (kind) {
    case "invalid_request":
      return "INVALID_REQUEST";
    case "validation":
      return "VALIDATION_FAILED";
    case "unauthorized":
      return "UNAUTHORIZED";
    case "not_found":
      return "NOT_FOUND";
    case "conflict":
      return "CONFLICT";
    case "rate_limited":
      return "RATE_LIMITED";
    case "service_unavailable":
      return "SERVICE_UNAVAILABLE";
    case "internal":
      return "INTERNAL_ERROR";
  }
}

/** Creates a TidekeepersError from an unknown thrown value. */
export function asTidekeepersError(err: unknown): TidekeepersError {
  if (err instanceof TidekeepersError) {
    return err;
  }
  if (err instanceof Error) {
    return new TidekeepersError(err.message, "internal", { cause: err });
  }
  return new TidekeepersError("internal error", "internal");
}

/** Maps a TidekeepersError to an HTTP status code. */
export function statusCodeFor(err: TidekeepersError): number {
  switch (err.kind) {
    case "invalid_request":
      return 400;
    case "validation":
      return 422;
    case "unauthorized":
      return 401;
    case "not_found":
      return 404;
    case "conflict":
      return 409;
    case "rate_limited":
      return 429;
    case "service_unavailable":
      return 503;
    case "internal":
      return 500;
  }
}

export function badRequest(
  message: string,
  options: ErrorOptions = {},
): TidekeepersError {
  return new TidekeepersError(message, "invalid_request", options);
}

export function validationFailed(
  message: string,
  field?: string,
  options: ErrorOptions = {},
): TidekeepersError {
  return new TidekeepersError(message, "validation", { ...options, field });
}

export function unauthorized(
  message = "unauthorized",
  options: ErrorOptions = {},
): TidekeepersError {
  return new TidekeepersError(message, "unauthorized", options);
}

export function notFound(
  message = "not found",
  options: ErrorOptions = {},
): TidekeepersError {
  return new TidekeepersError(message, "not_found", options);
}

export function conflict(
  message: string,
  options: ErrorOptions = {},
): TidekeepersError {
  return new TidekeepersError(message, "conflict", options);
}

export function rateLimited(
  message = "too many requests",
  options: ErrorOptions = {},
): TidekeepersError {
  return new TidekeepersError(message, "rate_limited", options);
}

export function internal(
  message = "internal error",
  options: ErrorOptions = {},
): TidekeepersError {
  return new TidekeepersError(message, "internal", options);
}

export function serviceUnavailable(
  message = "service unavailable",
  options: ErrorOptions = {},
): TidekeepersError {
  return new TidekeepersError(message, "service_unavailable", options);
}
