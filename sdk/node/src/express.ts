/**
 * SecureLens SDK - Express Middleware
 *
 * Middleware for integrating SecureLens AI Gate with Express.js applications.
 */

import type { Request, Response, NextFunction, RequestHandler } from 'express';
import { SecureLensClient } from './client.js';
import { AIGate } from './gate.js';
import type { UserContext, GateInterceptResponse, GateQueryResponse } from './types.js';
import { SecureLensPolicyError, isSecureLensError } from './errors.js';

/**
 * Configuration options for the SecureLens Express middleware.
 */
export interface SecureLensMiddlewareOptions {
  /** SecureLens client instance */
  client: SecureLensClient;
  /** Intercept and validate incoming requests */
  interceptRequests?: boolean;
  /** Audit outgoing responses */
  auditResponses?: boolean;
  /** Policies to apply */
  policies?: string[];
  /** Extract user context from request */
  getUserContext?: (req: Request) => UserContext | Promise<UserContext>;
  /** Extract query from request */
  getQuery?: (req: Request) => string | undefined;
  /** Custom error handler */
  onError?: (error: Error, req: Request, res: Response) => void;
  /** Called when a request is blocked by policy */
  onBlocked?: (result: GateInterceptResponse, req: Request, res: Response) => void;
  /** Skip middleware for certain requests */
  skip?: (req: Request) => boolean;
}

/**
 * Extended Express Request with SecureLens context.
 */
export interface SecureLensRequest extends Request {
  secureLens?: {
    interceptResult?: GateInterceptResponse;
    safeQuery?: string;
    auditId?: string;
    userContext?: UserContext;
  };
}

/**
 * Create SecureLens middleware for Express.
 *
 * @example
 * ```typescript
 * import express from 'express';
 * import { SecureLensClient, secureLensMiddleware } from '@securelens/sdk';
 *
 * const app = express();
 * const client = new SecureLensClient({ apiKey: 'sl_...' });
 *
 * // Apply to all /api/chat routes
 * app.use('/api/chat', secureLensMiddleware({
 *   client,
 *   interceptRequests: true,
 *   auditResponses: true,
 *   policies: ['redact_pii', 'block_sensitive_queries'],
 *   getUserContext: (req) => ({
 *     userId: req.user?.id,
 *     roles: req.user?.roles
 *   })
 * }));
 *
 * app.post('/api/chat', (req, res) => {
 *   // Access the safe query
 *   const safeQuery = req.secureLens?.safeQuery ?? req.body.query;
 *   // Process the query...
 * });
 * ```
 */
export function secureLensMiddleware(options: SecureLensMiddlewareOptions): RequestHandler {
  const gate = new AIGate(options.client);

  const defaultGetQuery = (req: Request): string | undefined => {
    return (
      (req.body as Record<string, unknown>)?.query as string ??
      (req.body as Record<string, unknown>)?.message as string ??
      (req.body as Record<string, unknown>)?.prompt as string ??
      (req.query as Record<string, unknown>)?.q as string
    );
  };

  const defaultOnError = (error: Error, _req: Request, res: Response): void => {
    if (isSecureLensError(error)) {
      res.status(error.statusCode ?? 500).json({
        error: error.code,
        message: error.message,
        requestId: error.requestId,
      });
    } else {
      res.status(500).json({
        error: 'INTERNAL_ERROR',
        message: 'An internal error occurred',
      });
    }
  };

  const defaultOnBlocked = (result: GateInterceptResponse, _req: Request, res: Response): void => {
    res.status(403).json({
      error: 'POLICY_BLOCKED',
      message: result.blockReason ?? 'Request blocked by policy',
      auditId: result.auditId,
      suggestedQuery: result.suggestedQuery,
    });
  };

  return async (req: Request, res: Response, next: NextFunction): Promise<void> => {
    // Skip if configured
    if (options.skip?.(req)) {
      next();
      return;
    }

    const slReq = req as SecureLensRequest;
    slReq.secureLens = {};

    try {
      // Get user context
      if (options.getUserContext) {
        slReq.secureLens.userContext = await options.getUserContext(req);
      }

      // Intercept request if enabled
      if (options.interceptRequests) {
        const getQuery = options.getQuery ?? defaultGetQuery;
        const query = getQuery(req);

        if (query) {
          const result = await gate.intercept({
            query,
            policies: options.policies,
            userContext: slReq.secureLens.userContext,
          });

          slReq.secureLens.interceptResult = result;
          slReq.secureLens.safeQuery = result.safeQuery;
          slReq.secureLens.auditId = result.auditId;

          if (!result.allowed) {
            const onBlocked = options.onBlocked ?? defaultOnBlocked;
            onBlocked(result, req, res);
            return;
          }
        }
      }

      // Wrap response to audit if enabled
      if (options.auditResponses) {
        const originalJson = res.json.bind(res);

        res.json = function (body: unknown): Response {
          // Fire and forget audit
          void auditResponse(gate, slReq, body);
          return originalJson(body);
        };
      }

      next();
    } catch (error) {
      const onError = options.onError ?? defaultOnError;
      onError(error as Error, req, res);
    }
  };
}

/**
 * Audit a response (fire and forget).
 */
async function auditResponse(
  gate: AIGate,
  req: SecureLensRequest,
  body: unknown
): Promise<void> {
  try {
    // Extract response text for auditing
    const responseText = extractResponseText(body);
    if (!responseText) return;

    // Classify the response
    await gate.classify(responseText);
  } catch {
    // Silently fail - don't block the response
  }
}

/**
 * Extract text from response body for auditing.
 */
function extractResponseText(body: unknown): string | undefined {
  if (typeof body === 'string') {
    return body;
  }

  if (body && typeof body === 'object') {
    const obj = body as Record<string, unknown>;
    return (
      (obj.response as string) ??
      (obj.message as string) ??
      (obj.content as string) ??
      (obj.text as string) ??
      (obj.answer as string)
    );
  }

  return undefined;
}

/**
 * Create a route handler that wraps the AI Gate query.
 *
 * @example
 * ```typescript
 * import { createGateHandler } from '@securelens/sdk/express';
 *
 * app.post('/api/chat', createGateHandler({
 *   client,
 *   llmProvider: 'openai',
 *   model: 'gpt-4',
 *   contextSource: 'qdrant',
 *   policies: ['redact_pii']
 * }));
 * ```
 */
export interface GateHandlerOptions {
  client: SecureLensClient;
  llmProvider?: string;
  model?: string;
  contextSource?: string;
  topK?: number;
  policies?: string[];
  getUserContext?: (req: Request) => UserContext | Promise<UserContext>;
  getQuery?: (req: Request) => string;
  formatResponse?: (result: GateQueryResponse) => unknown;
}

export function createGateHandler(options: GateHandlerOptions): RequestHandler {
  const gate = new AIGate(options.client);

  return async (req: Request, res: Response): Promise<void> => {
    try {
      const getQuery =
        options.getQuery ??
        ((r: Request) => (r.body as Record<string, unknown>)?.query as string);

      const query = getQuery(req);
      if (!query) {
        res.status(400).json({ error: 'Query is required' });
        return;
      }

      const userContext = options.getUserContext
        ? await options.getUserContext(req)
        : undefined;

      const result = await gate.query({
        query,
        llmProvider: options.llmProvider as 'openai' | 'anthropic' | 'azure' | 'bedrock' | 'cohere' | 'custom',
        model: options.model,
        contextSource: options.contextSource as 'qdrant' | 'pinecone' | 'weaviate' | 'milvus' | 'chroma' | 'custom',
        topK: options.topK,
        policies: options.policies,
        userContext,
      });

      const response = options.formatResponse ? options.formatResponse(result) : result;
      res.json(response);
    } catch (error) {
      if (error instanceof SecureLensPolicyError) {
        res.status(403).json({
          error: 'POLICY_BLOCKED',
          message: error.message,
          policyId: error.policyId,
          suggestedQuery: error.suggestedQuery,
        });
        return;
      }

      if (isSecureLensError(error)) {
        res.status(error.statusCode ?? 500).json({
          error: error.code,
          message: error.message,
        });
        return;
      }

      res.status(500).json({
        error: 'INTERNAL_ERROR',
        message: 'An internal error occurred',
      });
    }
  };
}
