/**
 * SecureLens SDK - Custom Error Classes
 *
 * Provides typed errors for different failure scenarios.
 */

/**
 * Base error class for all SecureLens SDK errors.
 */
export class SecureLensError extends Error {
  /** HTTP status code if applicable */
  readonly statusCode?: number;
  /** Error code for programmatic handling */
  readonly code: string;
  /** Request ID for support */
  readonly requestId?: string;
  /** Original error if wrapped */
  readonly cause?: Error;

  constructor(
    message: string,
    code: string,
    options?: {
      statusCode?: number;
      requestId?: string;
      cause?: Error;
    }
  ) {
    super(message);
    this.name = 'SecureLensError';
    this.code = code;
    this.statusCode = options?.statusCode;
    this.requestId = options?.requestId;
    this.cause = options?.cause;

    // Maintains proper stack trace for where error was thrown
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, this.constructor);
    }
  }

  toJSON(): Record<string, unknown> {
    return {
      name: this.name,
      message: this.message,
      code: this.code,
      statusCode: this.statusCode,
      requestId: this.requestId,
    };
  }
}

/**
 * Authentication error - invalid or expired API key.
 */
export class SecureLensAuthError extends SecureLensError {
  constructor(
    message = 'Authentication failed. Please check your API key.',
    options?: { requestId?: string; cause?: Error }
  ) {
    super(message, 'AUTH_ERROR', { statusCode: 401, ...options });
    this.name = 'SecureLensAuthError';
  }
}

/**
 * Rate limit exceeded error.
 */
export class SecureLensRateLimitError extends SecureLensError {
  /** Time in seconds until rate limit resets */
  readonly retryAfter?: number;
  /** Current rate limit */
  readonly limit?: number;
  /** Remaining requests */
  readonly remaining?: number;

  constructor(
    message = 'Rate limit exceeded. Please retry later.',
    options?: {
      requestId?: string;
      retryAfter?: number;
      limit?: number;
      remaining?: number;
      cause?: Error;
    }
  ) {
    super(message, 'RATE_LIMIT_ERROR', { statusCode: 429, ...options });
    this.name = 'SecureLensRateLimitError';
    this.retryAfter = options?.retryAfter;
    this.limit = options?.limit;
    this.remaining = options?.remaining;
  }

  toJSON(): Record<string, unknown> {
    return {
      ...super.toJSON(),
      retryAfter: this.retryAfter,
      limit: this.limit,
      remaining: this.remaining,
    };
  }
}

/**
 * Policy violation error - request blocked by governance policy.
 */
export class SecureLensPolicyError extends SecureLensError {
  /** Policy that was violated */
  readonly policyId?: string;
  /** Policy name */
  readonly policyName?: string;
  /** Detailed reason for violation */
  readonly reason?: string;
  /** Suggested alternative query */
  readonly suggestedQuery?: string;

  constructor(
    message = 'Request blocked by policy.',
    options?: {
      requestId?: string;
      policyId?: string;
      policyName?: string;
      reason?: string;
      suggestedQuery?: string;
      cause?: Error;
    }
  ) {
    super(message, 'POLICY_ERROR', { statusCode: 403, ...options });
    this.name = 'SecureLensPolicyError';
    this.policyId = options?.policyId;
    this.policyName = options?.policyName;
    this.reason = options?.reason;
    this.suggestedQuery = options?.suggestedQuery;
  }

  toJSON(): Record<string, unknown> {
    return {
      ...super.toJSON(),
      policyId: this.policyId,
      policyName: this.policyName,
      reason: this.reason,
      suggestedQuery: this.suggestedQuery,
    };
  }
}

/**
 * Connection error - network or server unreachable.
 */
export class SecureLensConnectionError extends SecureLensError {
  /** Whether the request can be retried */
  readonly retryable: boolean;

  constructor(
    message = 'Failed to connect to SecureLens API.',
    options?: {
      requestId?: string;
      retryable?: boolean;
      cause?: Error;
    }
  ) {
    super(message, 'CONNECTION_ERROR', { statusCode: 503, ...options });
    this.name = 'SecureLensConnectionError';
    this.retryable = options?.retryable ?? true;
  }

  toJSON(): Record<string, unknown> {
    return {
      ...super.toJSON(),
      retryable: this.retryable,
    };
  }
}

/**
 * Validation error - invalid request parameters.
 */
export class SecureLensValidationError extends SecureLensError {
  /** Field-level validation errors */
  readonly fieldErrors?: Record<string, string[]>;

  constructor(
    message = 'Invalid request parameters.',
    options?: {
      requestId?: string;
      fieldErrors?: Record<string, string[]>;
      cause?: Error;
    }
  ) {
    super(message, 'VALIDATION_ERROR', { statusCode: 400, ...options });
    this.name = 'SecureLensValidationError';
    this.fieldErrors = options?.fieldErrors;
  }

  toJSON(): Record<string, unknown> {
    return {
      ...super.toJSON(),
      fieldErrors: this.fieldErrors,
    };
  }
}

/**
 * Timeout error - request took too long.
 */
export class SecureLensTimeoutError extends SecureLensError {
  /** Timeout duration in milliseconds */
  readonly timeoutMs: number;

  constructor(
    message = 'Request timed out.',
    options?: {
      requestId?: string;
      timeoutMs?: number;
      cause?: Error;
    }
  ) {
    super(message, 'TIMEOUT_ERROR', { statusCode: 408, ...options });
    this.name = 'SecureLensTimeoutError';
    this.timeoutMs = options?.timeoutMs ?? 30000;
  }

  toJSON(): Record<string, unknown> {
    return {
      ...super.toJSON(),
      timeoutMs: this.timeoutMs,
    };
  }
}

/**
 * Server error - internal SecureLens API error.
 */
export class SecureLensServerError extends SecureLensError {
  constructor(
    message = 'Internal server error.',
    options?: {
      statusCode?: number;
      requestId?: string;
      cause?: Error;
    }
  ) {
    super(message, 'SERVER_ERROR', { statusCode: options?.statusCode ?? 500, ...options });
    this.name = 'SecureLensServerError';
  }
}

/**
 * Parse API error response and return appropriate error class.
 */
export function parseApiError(
  statusCode: number,
  body: unknown,
  requestId?: string
): SecureLensError {
  const errorBody = body as Record<string, unknown> | undefined;
  const message = (errorBody?.message as string) ?? (errorBody?.error as string) ?? 'Unknown error';

  switch (statusCode) {
    case 400:
      return new SecureLensValidationError(message, {
        requestId,
        fieldErrors: errorBody?.fieldErrors as Record<string, string[]>,
      });

    case 401:
      return new SecureLensAuthError(message, { requestId });

    case 403:
      return new SecureLensPolicyError(message, {
        requestId,
        policyId: errorBody?.policyId as string,
        policyName: errorBody?.policyName as string,
        reason: errorBody?.reason as string,
        suggestedQuery: errorBody?.suggestedQuery as string,
      });

    case 408:
      return new SecureLensTimeoutError(message, { requestId });

    case 429:
      return new SecureLensRateLimitError(message, {
        requestId,
        retryAfter: errorBody?.retryAfter as number,
        limit: errorBody?.limit as number,
        remaining: errorBody?.remaining as number,
      });

    case 503:
      return new SecureLensConnectionError(message, { requestId, retryable: true });

    default:
      if (statusCode >= 500) {
        return new SecureLensServerError(message, { statusCode, requestId });
      }
      return new SecureLensError(message, 'UNKNOWN_ERROR', { statusCode, requestId });
  }
}

/**
 * Check if an error is a SecureLens SDK error.
 */
export function isSecureLensError(error: unknown): error is SecureLensError {
  return error instanceof SecureLensError;
}

/**
 * Check if an error is retryable.
 */
export function isRetryableError(error: unknown): boolean {
  if (error instanceof SecureLensRateLimitError) {
    return true;
  }
  if (error instanceof SecureLensConnectionError) {
    return error.retryable;
  }
  if (error instanceof SecureLensTimeoutError) {
    return true;
  }
  if (error instanceof SecureLensServerError) {
    return error.statusCode !== undefined && error.statusCode >= 500 && error.statusCode < 600;
  }
  return false;
}
