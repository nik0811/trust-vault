# SecureLens SDK for Node.js

Enterprise-grade AI governance SDK for RAG applications. Secure your AI pipelines with classification, policy enforcement, and comprehensive audit trails.

[![npm version](https://badge.fury.io/js/@securelens%2Fsdk.svg)](https://www.npmjs.com/package/@securelens/sdk)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0+-blue.svg)](https://www.typescriptlang.org/)
[![Node.js](https://img.shields.io/badge/Node.js-18+-green.svg)](https://nodejs.org/)

## Features

- **AI Gate** - Intercept, classify, and govern all queries to your RAG system
- **Real-time Classification** - Detect 60+ PII types, PHI, PCI, and custom patterns
- **Policy Enforcement** - Block, redact, or mask sensitive data based on policies
- **Full Audit Trail** - OpenLineage-compatible audit events for compliance
- **Express Middleware** - Drop-in middleware for Express.js applications
- **LangChain Integration** - Callback handlers for LangChain.js pipelines
- **Streaming Support** - Real-time streaming with governance applied
- **TypeScript First** - Full type definitions for excellent DX

## Installation

```bash
npm install @securelens/sdk
```

## Quick Start

```typescript
import { SecureLensClient, AIGate } from '@securelens/sdk';

// Initialize the client
const client = new SecureLensClient({
  apiKey: 'sl_your_api_key_here',
  baseUrl: 'https://api.securelens.ai'
});

// Create the AI Gate
const gate = new AIGate(client);

// Intercept a query
const result = await gate.intercept({
  query: "What is John's salary?",
  policies: ['redact_pii', 'block_salary_queries']
});

if (result.allowed) {
  console.log('Safe query:', result.safeQuery);
} else {
  console.log('Blocked:', result.blockReason);
}
```

## Core Concepts

### SecureLensClient

The main client for API communication:

```typescript
const client = new SecureLensClient({
  apiKey: 'sl_...',           // Required: Your API key
  baseUrl: 'https://...',     // Optional: API endpoint
  tenantId: 'tenant_123',     // Optional: Multi-tenant ID
  timeout: 30000,             // Optional: Request timeout (ms)
  retries: 3,                 // Optional: Retry attempts
  debug: false                // Optional: Enable debug logging
});
```

### AIGate

The AI Gate provides three main operations:

#### 1. Intercept - Validate queries before processing

```typescript
const result = await gate.intercept({
  query: "What is John Smith's SSN?",
  policies: ['redact_pii', 'block_ssn_queries'],
  userContext: {
    userId: 'user_123',
    roles: ['analyst'],
    clearanceLevel: 'confidential'
  }
});

// result.allowed - Whether the query is permitted
// result.safeQuery - The sanitized query
// result.classifications - PII/sensitive data found
// result.auditId - Audit trail reference
```

#### 2. Query - Full governed RAG query

```typescript
const response = await gate.query({
  query: "What are the Q3 projections?",
  contextSource: 'qdrant',      // Vector DB: qdrant, pinecone, weaviate, etc.
  llmProvider: 'openai',        // LLM: openai, anthropic, azure, etc.
  model: 'gpt-4',
  topK: 5,
  policies: ['financial_data_access']
});

// response.response - The LLM response
// response.safeQuery - Query that was sent
// response.tokensUsed - Token usage stats
// response.context - Retrieved documents
// response.auditId - Audit trail reference
```

#### 3. Embed - Governed document embedding

```typescript
const result = await gate.embed({
  documents: ['John Smith earns $150,000...'],
  metadata: [{ source: 'hr_docs' }],
  vectorDb: 'qdrant',
  classifyBeforeEmbed: true,
  chunkSize: 512,
  chunkOverlap: 50
});

// result.documentsEmbedded - Count of documents
// result.chunksCreated - Count of chunks
// result.classifications - Sensitive data found
// result.documentIds - IDs in vector DB
```

### Streaming

For real-time responses with governance:

```typescript
await gate.queryStream(
  {
    query: "Summarize the report",
    llmProvider: 'openai',
    model: 'gpt-4',
    stream: true
  },
  (chunk) => {
    if (chunk.type === 'content') {
      process.stdout.write(chunk.content);
    } else if (chunk.type === 'done') {
      console.log('\nAudit ID:', chunk.finalResponse?.auditId);
    }
  }
);
```

## Express.js Integration

### Middleware

```typescript
import express from 'express';
import { SecureLensClient } from '@securelens/sdk';
import { secureLensMiddleware } from '@securelens/sdk/express';

const app = express();
const client = new SecureLensClient({ apiKey: 'sl_...' });

app.use('/api/chat', secureLensMiddleware({
  client,
  interceptRequests: true,
  auditResponses: true,
  policies: ['redact_pii'],
  getUserContext: (req) => ({
    userId: req.user?.id,
    roles: req.user?.roles
  })
}));

app.post('/api/chat', (req, res) => {
  // req.secureLens.safeQuery - The sanitized query
  // req.secureLens.auditId - Audit reference
  res.json({ response: '...' });
});
```

### Gate Handler

```typescript
import { createGateHandler } from '@securelens/sdk/express';

app.post('/api/ai/query', createGateHandler({
  client,
  llmProvider: 'openai',
  model: 'gpt-4',
  contextSource: 'qdrant',
  policies: ['redact_pii']
}));
```

## LangChain.js Integration

### Callback Handler

```typescript
import { SecureLensCallbackHandler } from '@securelens/sdk/langchain';

const handler = new SecureLensCallbackHandler(gate, {
  policies: ['redact_pii'],
  blockOnPolicyViolation: true,
  classifyRetrievedDocs: true,
  onClassification: (classifications, source) => {
    console.log(`Found ${classifications.length} in ${source}`);
  }
});

// Use with LangChain
const chain = RetrievalQAChain.fromLLM(llm, retriever, {
  callbacks: [handler]
});
```

### Secure Retriever

```typescript
import { createSecureLensRetriever } from '@securelens/sdk/langchain';

const secureRetriever = createSecureLensRetriever(gate, baseRetriever, {
  classifyDocuments: true,
  filterSensitive: true,
  maxSensitivity: 'confidential'
});
```

## Error Handling

The SDK provides typed errors for different scenarios:

```typescript
import {
  SecureLensAuthError,
  SecureLensRateLimitError,
  SecureLensPolicyError,
  SecureLensConnectionError,
  isSecureLensError,
  isRetryableError
} from '@securelens/sdk';

try {
  await gate.query({ query: '...' });
} catch (error) {
  if (error instanceof SecureLensPolicyError) {
    console.log('Blocked by policy:', error.policyName);
    console.log('Suggestion:', error.suggestedQuery);
  } else if (error instanceof SecureLensRateLimitError) {
    console.log('Rate limited, retry after:', error.retryAfter);
  } else if (isRetryableError(error)) {
    // Implement retry logic
  }
}
```

## TypeScript Types

Full TypeScript support with comprehensive types:

```typescript
import type {
  SecureLensConfig,
  GateQueryRequest,
  GateQueryResponse,
  Classification,
  SensitivityLevel,
  UserContext,
  AuditEvent
} from '@securelens/sdk';
```

## Configuration

### Environment Variables

```bash
SECURELENS_API_KEY=sl_your_api_key
SECURELENS_BASE_URL=https://api.securelens.ai
SECURELENS_TENANT_ID=your_tenant_id
```

### Vector Database Support

- Qdrant
- Pinecone
- Weaviate
- Milvus
- Chroma
- Custom (bring your own)

### LLM Provider Support

- OpenAI
- Anthropic
- Azure OpenAI
- AWS Bedrock
- Cohere
- Custom (bring your own)

## Examples

See the [examples](./examples) directory for complete working examples:

- [Basic Query](./examples/basic-query.ts) - Core SDK usage
- [Express Middleware](./examples/express-middleware.ts) - Express.js integration
- [LangChain Integration](./examples/langchain-integration.ts) - LangChain.js usage

## API Reference

### SecureLensClient

| Method | Description |
|--------|-------------|
| `get<T>(path, options?)` | Make a GET request |
| `post<T>(path, body?, options?)` | Make a POST request |
| `put<T>(path, body?, options?)` | Make a PUT request |
| `delete<T>(path, options?)` | Make a DELETE request |
| `health()` | Check API health status |

### AIGate

| Method | Description |
|--------|-------------|
| `intercept(request)` | Intercept and validate a query |
| `query(request)` | Execute a governed RAG query |
| `queryStream(request, callback)` | Stream a governed query |
| `embed(request)` | Embed documents with classification |
| `classify(text)` | Classify text for sensitive data |
| `classifyBatch(texts)` | Batch classify multiple texts |
| `getAuditEvents(options?)` | Query audit events |
| `getAuditEvent(auditId)` | Get a specific audit event |

## Requirements

- Node.js 18+
- TypeScript 5.0+ (for TypeScript users)

## License

MIT

## Support

- Documentation: https://securelens.ai/docs
- Issues: https://github.com/securelens/sdk-node/issues
- Email: support@securelens.ai
