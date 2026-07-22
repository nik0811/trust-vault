/**
 * SecureLens SDK - HTTP Client
 *
 * Core HTTP client for communicating with the SecureLens API.
 */

import {
  SecureLensConfig,
  RequestOptions,
  HealthStatus,
} from './types.js';
import {
  SecureLensError,
  SecureLensConnectionError,
  SecureLensTimeoutError,
  parseApiError,
  isRetryableError,
} from './errors.js';
import { createAuthContext, maskApiKey, type AuthContext } from './auth.js';

const DEFAULT_BASE_URL = 'https://api.securelens.ai';
const DEFAULT_TIMEOUT = 30000;
const DEFAULT_RETRIES = 3;
const RETRY_DELAY_BASE = 1000;

/**
 * SecureLens API client for making authenticated requests.
 */
export class SecureLensClient {
  private readonly config: Required<
    Pick<SecureLensConfig, 'baseUrl' | 'timeout' | 'retries' | 'debug'>
  > &
    SecureLensConfig;
  private readonly auth: AuthContext;

  constructor(config: SecureLensConfig) {
    this.config = {
      ...config,
      baseUrl: config.baseUrl ?? DEFAULT_BASE_URL,
      timeout: config.timeout ?? DEFAULT_TIMEOUT,
      retries: config.retries ?? DEFAULT_RETRIES,
      debug: config.debug ?? false,
    };

    this.auth = createAuthContext(config.apiKey, config.tenantId);

    if (this.config.debug) {
      this.log('Client initialized', {
        baseUrl: this.config.baseUrl,
        apiKey: maskApiKey(config.apiKey),
        tenantId: config.tenantId,
      });
    }
  }

  /**
   * Make a GET request to the API.
   */
  async get<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('GET', path, undefined, options);
  }

  /**
   * Make a POST request to the API.
   */
  async post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>('POST', path, body, options);
  }

  /**
   * Make a PUT request to the API.
   */
  async put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>('PUT', path, body, options);
  }

  /**
   * Make a DELETE request to the API.
   */
  async delete<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>('DELETE', path, undefined, options);
  }

  /**
   * Check API health status.
   */
  async health(): Promise<HealthStatus> {
    return this.get<HealthStatus>('/health');
  }

  /**
   * Get the base URL.
   */
  getBaseUrl(): string {
    return this.config.baseUrl;
  }

  /**
   * Get the tenant ID.
   */
  getTenantId(): string | undefined {
    return this.config.tenantId;
  }

  /**
   * Core request method with retry logic.
   */
  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: RequestOptions
  ): Promise<T> {
    const url = `${this.config.baseUrl}${path}`;
    const timeout = options?.timeout ?? this.config.timeout;

    let lastError: Error | undefined;

    for (let attempt = 0; attempt <= this.config.retries; attempt++) {
      try {
        if (attempt > 0) {
          const delay = RETRY_DELAY_BASE * Math.pow(2, attempt - 1);
          if (this.config.debug) {
            this.log(`Retry attempt ${attempt} after ${delay}ms`);
          }
          await this.sleep(delay);
        }

        const response = await this.executeRequest(method, url, body, timeout, options);
        return response as T;
      } catch (error) {
        lastError = error as Error;

        if (!isRetryableError(error) || attempt === this.config.retries) {
          throw error;
        }

        if (this.config.debug) {
          this.log(`Request failed, will retry`, { error: (error as Error).message });
        }
      }
    }

    throw lastError ?? new SecureLensError('Request failed', 'UNKNOWN_ERROR');
  }

  /**
   * Execute a single HTTP request.
   */
  private async executeRequest(
    method: string,
    url: string,
    body: unknown,
    timeout: number,
    options?: RequestOptions
  ): Promise<unknown> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    const headers: Record<string, string> = {
      ...this.auth.headers,
      ...options?.headers,
    };

    const requestId = this.generateRequestId();
    headers['X-Request-ID'] = requestId;

    if (this.config.debug) {
      this.log(`${method} ${url}`, { requestId });
    }

    try {
      const response = await fetch(url, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
        signal: options?.signal ?? controller.signal,
      });

      clearTimeout(timeoutId);

      const responseBody = await this.parseResponse(response);

      if (!response.ok) {
        throw parseApiError(response.status, responseBody, requestId);
      }

      if (this.config.debug) {
        this.log(`Response ${response.status}`, { requestId });
      }

      return responseBody;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof SecureLensError) {
        throw error;
      }

      if ((error as Error).name === 'AbortError') {
        throw new SecureLensTimeoutError(`Request timed out after ${timeout}ms`, {
          requestId,
          timeoutMs: timeout,
        });
      }

      throw new SecureLensConnectionError(`Failed to connect: ${(error as Error).message}`, {
        requestId,
        retryable: true,
        cause: error as Error,
      });
    }
  }

  /**
   * Parse response body as JSON.
   */
  private async parseResponse(response: Response): Promise<unknown> {
    const contentType = response.headers.get('content-type');

    if (contentType?.includes('application/json')) {
      return response.json();
    }

    const text = await response.text();
    try {
      return JSON.parse(text);
    } catch {
      return { message: text };
    }
  }

  /**
   * Generate a unique request ID.
   */
  private generateRequestId(): string {
    return `req_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`;
  }

  /**
   * Sleep for a given duration.
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Log a debug message.
   */
  private log(message: string, data?: Record<string, unknown>): void {
    const timestamp = new Date().toISOString();
    const prefix = '[SecureLens]';

    if (data) {
      console.log(`${timestamp} ${prefix} ${message}`, data);
    } else {
      console.log(`${timestamp} ${prefix} ${message}`);
    }
  }
}
