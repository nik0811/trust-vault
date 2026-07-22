/**
 * SecureLens SDK - Authentication Module
 *
 * Handles API key validation and authentication headers.
 */

import { SecureLensAuthError } from './errors.js';

const API_KEY_PREFIX = 'sl_';
const API_KEY_MIN_LENGTH = 32;

/**
 * Validates the format of a SecureLens API key.
 */
export function validateApiKey(apiKey: string): void {
  if (!apiKey) {
    throw new SecureLensAuthError('API key is required.');
  }

  if (!apiKey.startsWith(API_KEY_PREFIX)) {
    throw new SecureLensAuthError(
      `Invalid API key format. API key must start with "${API_KEY_PREFIX}".`
    );
  }

  if (apiKey.length < API_KEY_MIN_LENGTH) {
    throw new SecureLensAuthError(
      `Invalid API key format. API key must be at least ${API_KEY_MIN_LENGTH} characters.`
    );
  }
}

/**
 * Creates authentication headers for API requests.
 */
export function createAuthHeaders(
  apiKey: string,
  tenantId?: string
): Record<string, string> {
  const headers: Record<string, string> = {
    Authorization: `Bearer ${apiKey}`,
    'Content-Type': 'application/json',
  };

  if (tenantId) {
    headers['X-Tenant-ID'] = tenantId;
  }

  return headers;
}

/**
 * Masks an API key for safe logging.
 */
export function maskApiKey(apiKey: string): string {
  if (!apiKey || apiKey.length < 12) {
    return '***';
  }
  return `${apiKey.slice(0, 6)}...${apiKey.slice(-4)}`;
}

/**
 * Authentication context for requests.
 */
export interface AuthContext {
  apiKey: string;
  tenantId?: string;
  headers: Record<string, string>;
}

/**
 * Creates an authentication context.
 */
export function createAuthContext(apiKey: string, tenantId?: string): AuthContext {
  validateApiKey(apiKey);

  return {
    apiKey,
    tenantId,
    headers: createAuthHeaders(apiKey, tenantId),
  };
}
