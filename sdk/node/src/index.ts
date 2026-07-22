/**
 * SecureLens SDK for Node.js
 *
 * Enterprise-grade AI governance for RAG applications.
 *
 * @packageDocumentation
 */

// Core exports
import { SecureLensClient as _SecureLensClient } from './client.js';
import { AIGate as _AIGate } from './gate.js';

export { SecureLensClient } from './client.js';
export { AIGate } from './gate.js';

// Authentication
export {
  validateApiKey,
  createAuthHeaders,
  maskApiKey,
  createAuthContext,
  type AuthContext,
} from './auth.js';

// Error classes
export {
  SecureLensError,
  SecureLensAuthError,
  SecureLensRateLimitError,
  SecureLensPolicyError,
  SecureLensConnectionError,
  SecureLensValidationError,
  SecureLensTimeoutError,
  SecureLensServerError,
  parseApiError,
  isSecureLensError,
  isRetryableError,
} from './errors.js';

// Types
export type {
  // Configuration
  SecureLensConfig,
  RequestOptions,

  // Vector DB
  VectorDbProvider,
  VectorDbConfig,

  // LLM
  LLMProvider,
  LLMConfig,

  // Classification
  SensitivityLevel,
  ClassificationType,
  Classification,
  ClassificationResult,

  // Gate requests/responses
  GateInterceptRequest,
  GateInterceptResponse,
  GateQueryRequest,
  GateQueryResponse,
  GateEmbedRequest,
  GateEmbedResponse,

  // Supporting types
  UserContext,
  AppliedPolicy,
  TokenUsage,
  ContextDocument,

  // Audit
  AuditEvent,
  AuditQueryOptions,

  // Streaming
  StreamChunk,
  StreamCallback,

  // Health
  HealthStatus,
  ComponentStatus,
} from './types.js';

// Version
export const VERSION = '1.0.0';

/**
 * Create a new SecureLens client with the given configuration.
 *
 * @example
 * ```typescript
 * import { createClient } from '@securelens/sdk';
 *
 * const client = createClient({
 *   apiKey: 'sl_...',
 *   baseUrl: 'https://api.securelens.ai'
 * });
 * ```
 */
export function createClient(
  config: import('./types.js').SecureLensConfig
): _SecureLensClient {
  return new _SecureLensClient(config);
}

/**
 * Create a new AI Gate instance.
 *
 * @example
 * ```typescript
 * import { createClient, createGate } from '@securelens/sdk';
 *
 * const client = createClient({ apiKey: 'sl_...' });
 * const gate = createGate(client);
 *
 * const result = await gate.intercept({ query: 'Hello' });
 * ```
 */
export function createGate(client: _SecureLensClient): _AIGate {
  return new _AIGate(client);
}
