/**
 * Express Middleware Example
 *
 * Demonstrates how to integrate SecureLens with an Express.js application
 * to add AI governance to your chat API endpoints.
 *
 * Run: npx ts-node examples/express-middleware.ts
 */

import express, { Request, Response } from 'express';
import { SecureLensClient, AIGate } from '../src';
import {
  secureLensMiddleware,
  createGateHandler,
  SecureLensRequest,
} from '../src/express';

// Initialize Express app
const app = express();
app.use(express.json());

// Initialize SecureLens client
const client = new SecureLensClient({
  apiKey: process.env.SECURELENS_API_KEY ?? 'sl_test_key_here',
  baseUrl: process.env.SECURELENS_BASE_URL ?? 'https://api.securelens.ai',
  debug: true,
});

const gate = new AIGate(client);

// ============================================================================
// Example 1: Basic middleware for query interception
// ============================================================================

app.use(
  '/api/chat',
  secureLensMiddleware({
    client,
    interceptRequests: true,
    auditResponses: true,
    policies: ['redact_pii', 'block_sensitive_queries'],

    // Extract user context from request (e.g., from JWT)
    getUserContext: (req) => ({
      userId: (req as Request & { user?: { id: string } }).user?.id ?? 'anonymous',
      roles: ['user'],
    }),

    // Custom query extraction
    getQuery: (req) =>
      (req.body as { message?: string; query?: string })?.message ??
      (req.body as { message?: string; query?: string })?.query,

    // Custom blocked handler
    onBlocked: (result, _req, res) => {
      res.status(403).json({
        error: 'QUERY_BLOCKED',
        message: 'Your query was blocked by our security policies.',
        reason: result.blockReason,
        suggestion: result.suggestedQuery,
        auditId: result.auditId,
      });
    },

    // Skip health check endpoints
    skip: (req) => req.path === '/health',
  })
);

// Chat endpoint with middleware protection
app.post('/api/chat', async (req: Request, res: Response) => {
  const slReq = req as SecureLensRequest;

  // Access the safe query (PII redacted, etc.)
  const safeQuery = slReq.secureLens?.safeQuery ?? (req.body as { query?: string })?.query;

  // Your existing chat logic here
  // The query has already been validated and sanitized by SecureLens

  res.json({
    response: `Processed query: ${safeQuery}`,
    auditId: slReq.secureLens?.auditId,
    classifications: slReq.secureLens?.interceptResult?.classifications ?? [],
  });
});

// ============================================================================
// Example 2: Full AI Gate handler
// ============================================================================

app.post(
  '/api/ai/query',
  createGateHandler({
    client,
    llmProvider: 'openai',
    model: 'gpt-4',
    contextSource: 'qdrant',
    topK: 5,
    policies: ['redact_pii', 'financial_data_access'],

    getUserContext: (req) => ({
      userId: (req as Request & { user?: { id: string } }).user?.id,
      roles: (req as Request & { user?: { roles: string[] } }).user?.roles ?? [],
      clearanceLevel: 'confidential',
    }),

    // Custom response formatting
    formatResponse: (result) => ({
      answer: result.response,
      sources: result.context?.map((doc) => ({
        id: doc.id,
        score: doc.score,
        preview: doc.content.slice(0, 100) + '...',
      })),
      metadata: {
        auditId: result.auditId,
        tokensUsed: result.tokensUsed.totalTokens,
        processingTime: result.processingTimeMs,
      },
    }),
  })
);

// ============================================================================
// Example 3: Manual integration with more control
// ============================================================================

app.post('/api/advanced/chat', async (req: Request, res: Response) => {
  const { query, sessionId } = req.body as { query: string; sessionId?: string };

  try {
    // Step 1: Intercept and validate the query
    const interceptResult = await gate.intercept({
      query,
      policies: ['redact_pii', 'block_salary_queries', 'financial_data_access'],
      userContext: {
        userId: 'user_123',
        roles: ['analyst'],
        department: 'finance',
      },
      sessionId,
    });

    if (!interceptResult.allowed) {
      res.status(403).json({
        error: 'POLICY_VIOLATION',
        message: interceptResult.blockReason,
        suggestedQuery: interceptResult.suggestedQuery,
        auditId: interceptResult.auditId,
      });
      return;
    }

    // Step 2: Execute the governed query
    const queryResult = await gate.query({
      query: interceptResult.safeQuery,
      contextSource: 'qdrant',
      llmProvider: 'openai',
      model: 'gpt-4',
      topK: 5,
      sessionId,
    });

    // Step 3: Return the response
    res.json({
      response: queryResult.response,
      metadata: {
        originalQuery: query,
        safeQuery: queryResult.safeQuery,
        classifications: queryResult.classifications,
        policies: queryResult.appliedPolicies,
        auditId: queryResult.auditId,
        tokens: queryResult.tokensUsed,
      },
    });
  } catch (error) {
    console.error('Chat error:', error);
    res.status(500).json({
      error: 'INTERNAL_ERROR',
      message: 'An error occurred processing your request',
    });
  }
});

// ============================================================================
// Example 4: Document embedding endpoint
// ============================================================================

app.post('/api/documents/embed', async (req: Request, res: Response) => {
  const { documents, metadata } = req.body as {
    documents: string[];
    metadata?: Record<string, unknown>[];
  };

  try {
    const result = await gate.embed({
      documents,
      metadata,
      vectorDb: 'qdrant',
      classifyBeforeEmbed: true,
      policies: ['redact_pii_in_embeddings'],
      chunkSize: 512,
      chunkOverlap: 50,
    });

    res.json({
      success: true,
      documentsEmbedded: result.documentsEmbedded,
      chunksCreated: result.chunksCreated,
      documentIds: result.documentIds,
      auditId: result.auditId,
      classifications: result.classifications,
    });
  } catch (error) {
    console.error('Embedding error:', error);
    res.status(500).json({
      error: 'EMBEDDING_FAILED',
      message: 'Failed to embed documents',
    });
  }
});

// ============================================================================
// Example 5: Streaming endpoint
// ============================================================================

app.post('/api/chat/stream', async (req: Request, res: Response) => {
  const { query } = req.body as { query: string };

  // Set up SSE headers
  res.setHeader('Content-Type', 'text/event-stream');
  res.setHeader('Cache-Control', 'no-cache');
  res.setHeader('Connection', 'keep-alive');

  try {
    await gate.queryStream(
      {
        query,
        llmProvider: 'openai',
        model: 'gpt-4',
        contextSource: 'qdrant',
        stream: true,
      },
      (chunk) => {
        res.write(`data: ${JSON.stringify(chunk)}\n\n`);
      }
    );

    res.write('data: [DONE]\n\n');
    res.end();
  } catch (error) {
    res.write(`data: ${JSON.stringify({ type: 'error', error: 'Stream failed' })}\n\n`);
    res.end();
  }
});

// Health check
app.get('/health', (_req, res) => {
  res.json({ status: 'ok' });
});

// Start server
const PORT = process.env.PORT ?? 3000;
app.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
  console.log('\nEndpoints:');
  console.log('  POST /api/chat - Basic chat with middleware');
  console.log('  POST /api/ai/query - Full AI Gate query');
  console.log('  POST /api/advanced/chat - Advanced manual integration');
  console.log('  POST /api/documents/embed - Document embedding');
  console.log('  POST /api/chat/stream - Streaming chat');
});
