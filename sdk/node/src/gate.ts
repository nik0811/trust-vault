/**
 * SecureLens SDK - AI Gate
 *
 * Main interface for AI governance operations: intercept, query, and embed.
 */

import { SecureLensClient } from './client.js';
import {
  GateInterceptRequest,
  GateInterceptResponse,
  GateQueryRequest,
  GateQueryResponse,
  GateEmbedRequest,
  GateEmbedResponse,
  ClassificationResult,
  AuditEvent,
  AuditQueryOptions,
  StreamChunk,
  StreamCallback,
  VectorDbConfig,
  LLMConfig,
  VectorDbProvider,
  LLMProvider,
} from './types.js';
import { SecureLensValidationError } from './errors.js';

/**
 * AI Gate - Secure gateway for RAG applications.
 *
 * Provides methods to intercept queries, execute governed queries,
 * and embed documents with classification.
 */
export class AIGate {
  private readonly client: SecureLensClient;

  constructor(client: SecureLensClient) {
    this.client = client;
  }

  /**
   * Intercept and analyze a query before sending to LLM.
   *
   * Use this to check if a query is safe and apply governance policies
   * without actually executing the query.
   *
   * @example
   * ```typescript
   * const result = await gate.intercept({
   *   query: "What is John's salary?",
   *   policies: ['redact_pii', 'block_salary_queries']
   * });
   *
   * if (result.allowed) {
   *   // Proceed with the safe query
   *   console.log(result.safeQuery);
   * } else {
   *   console.log('Query blocked:', result.blockReason);
   * }
   * ```
   */
  async intercept(request: GateInterceptRequest): Promise<GateInterceptResponse> {
    this.validateInterceptRequest(request);

    const payload = this.normalizeInterceptRequest(request);
    return this.client.post<GateInterceptResponse>('/api/v1/gate/intercept', payload);
  }

  /**
   * Execute a full governed query through the AI Gate.
   *
   * This method:
   * 1. Intercepts and classifies the query
   * 2. Retrieves context from the vector database
   * 3. Applies governance policies
   * 4. Sends to the LLM
   * 5. Classifies and governs the response
   * 6. Returns the safe response with audit trail
   *
   * @example
   * ```typescript
   * const response = await gate.query({
   *   query: "What are the Q3 revenue projections?",
   *   contextSource: 'qdrant',
   *   llmProvider: 'openai',
   *   model: 'gpt-4',
   *   topK: 5,
   *   policies: ['redact_pii', 'financial_data_access']
   * });
   *
   * console.log(response.response);
   * console.log('Tokens used:', response.tokensUsed.totalTokens);
   * ```
   */
  async query(request: GateQueryRequest): Promise<GateQueryResponse> {
    this.validateQueryRequest(request);

    const payload = this.normalizeQueryRequest(request);
    return this.client.post<GateQueryResponse>('/api/v1/gate/query', payload);
  }

  /**
   * Execute a streaming governed query.
   *
   * Streams the response in chunks, allowing real-time display
   * while still applying governance policies.
   *
   * @example
   * ```typescript
   * await gate.queryStream(
   *   {
   *     query: "Summarize the quarterly report",
   *     llmProvider: 'openai',
   *     model: 'gpt-4',
   *     stream: true
   *   },
   *   (chunk) => {
   *     if (chunk.type === 'content') {
   *       process.stdout.write(chunk.content);
   *     } else if (chunk.type === 'done') {
   *       console.log('\nAudit ID:', chunk.finalResponse?.auditId);
   *     }
   *   }
   * );
   * ```
   */
  async queryStream(
    request: GateQueryRequest,
    callback: StreamCallback
  ): Promise<GateQueryResponse> {
    this.validateQueryRequest(request);

    const payload = this.normalizeQueryRequest({ ...request, stream: true });

    const response = await fetch(`${this.client.getBaseUrl()}/api/v1/gate/query/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.getApiKey()}`,
        ...(this.client.getTenantId() && { 'X-Tenant-ID': this.client.getTenantId() }),
      },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new SecureLensValidationError(error.message ?? 'Stream request failed');
    }

    const reader = response.body?.getReader();
    if (!reader) {
      throw new SecureLensValidationError('Response body is not readable');
    }

    const decoder = new TextDecoder();
    let finalResponse: GateQueryResponse | undefined;

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const text = decoder.decode(value, { stream: true });
        const lines = text.split('\n').filter((line) => line.startsWith('data: '));

        for (const line of lines) {
          const data = line.slice(6);
          if (data === '[DONE]') continue;

          try {
            const chunk = JSON.parse(data) as StreamChunk;
            callback(chunk);

            if (chunk.type === 'done' && chunk.finalResponse) {
              finalResponse = chunk.finalResponse;
            }
          } catch {
            // Skip malformed chunks
          }
        }
      }
    } finally {
      reader.releaseLock();
    }

    if (!finalResponse) {
      throw new SecureLensValidationError('Stream ended without final response');
    }

    return finalResponse;
  }

  /**
   * Embed documents into a vector database with classification.
   *
   * Optionally classifies documents before embedding to enable
   * governance-aware retrieval.
   *
   * @example
   * ```typescript
   * const result = await gate.embed({
   *   documents: [
   *     'John Smith earns $150,000 annually...',
   *     'The company was founded in 2020...'
   *   ],
   *   metadata: [
   *     { source: 'hr_docs', department: 'hr' },
   *     { source: 'company_info', department: 'general' }
   *   ],
   *   vectorDb: 'qdrant',
   *   classifyBeforeEmbed: true,
   *   chunkSize: 512,
   *   chunkOverlap: 50
   * });
   *
   * console.log('Embedded:', result.documentsEmbedded);
   * console.log('Chunks:', result.chunksCreated);
   * ```
   */
  async embed(request: GateEmbedRequest): Promise<GateEmbedResponse> {
    this.validateEmbedRequest(request);

    const payload = this.normalizeEmbedRequest(request);
    return this.client.post<GateEmbedResponse>('/api/v1/gate/embed', payload);
  }

  /**
   * Classify text without embedding or querying.
   *
   * Useful for pre-flight checks or standalone classification.
   *
   * @example
   * ```typescript
   * const result = await gate.classify('John Smith, SSN: 123-45-6789');
   * console.log(result.classifications);
   * // [{ type: 'pii', label: 'person_name', ... }, { type: 'pii', label: 'ssn', ... }]
   * ```
   */
  async classify(text: string): Promise<ClassificationResult> {
    if (!text || typeof text !== 'string') {
      throw new SecureLensValidationError('Text is required for classification');
    }

    return this.client.post<ClassificationResult>('/api/v1/classify', { text });
  }

  /**
   * Classify multiple texts in batch.
   *
   * More efficient than calling classify() multiple times.
   */
  async classifyBatch(texts: string[]): Promise<ClassificationResult[]> {
    if (!Array.isArray(texts) || texts.length === 0) {
      throw new SecureLensValidationError('Texts array is required for batch classification');
    }

    return this.client.post<ClassificationResult[]>('/api/v1/classify/batch', { texts });
  }

  /**
   * Get audit events for queries processed through the gate.
   */
  async getAuditEvents(options?: AuditQueryOptions): Promise<AuditEvent[]> {
    const params = new URLSearchParams();

    if (options?.startDate) {
      params.set('startDate', options.startDate.toISOString());
    }
    if (options?.endDate) {
      params.set('endDate', options.endDate.toISOString());
    }
    if (options?.userId) {
      params.set('userId', options.userId);
    }
    if (options?.eventType) {
      params.set('eventType', options.eventType);
    }
    if (options?.policyId) {
      params.set('policyId', options.policyId);
    }
    if (options?.limit) {
      params.set('limit', options.limit.toString());
    }
    if (options?.offset) {
      params.set('offset', options.offset.toString());
    }

    const queryString = params.toString();
    const path = queryString ? `/api/v1/audit?${queryString}` : '/api/v1/audit';

    return this.client.get<AuditEvent[]>(path);
  }

  /**
   * Get a specific audit event by ID.
   */
  async getAuditEvent(auditId: string): Promise<AuditEvent> {
    if (!auditId) {
      throw new SecureLensValidationError('Audit ID is required');
    }

    return this.client.get<AuditEvent>(`/api/v1/audit/${auditId}`);
  }

  // ============================================================================
  // Private Methods
  // ============================================================================

  private validateInterceptRequest(request: GateInterceptRequest): void {
    if (!request.query || typeof request.query !== 'string') {
      throw new SecureLensValidationError('Query is required', {
        fieldErrors: { query: ['Query must be a non-empty string'] },
      });
    }
  }

  private validateQueryRequest(request: GateQueryRequest): void {
    if (!request.query || typeof request.query !== 'string') {
      throw new SecureLensValidationError('Query is required', {
        fieldErrors: { query: ['Query must be a non-empty string'] },
      });
    }

    if (request.topK !== undefined && (request.topK < 1 || request.topK > 100)) {
      throw new SecureLensValidationError('Invalid topK value', {
        fieldErrors: { topK: ['topK must be between 1 and 100'] },
      });
    }
  }

  private validateEmbedRequest(request: GateEmbedRequest): void {
    if (!Array.isArray(request.documents) || request.documents.length === 0) {
      throw new SecureLensValidationError('Documents are required', {
        fieldErrors: { documents: ['Documents must be a non-empty array'] },
      });
    }

    if (!request.vectorDb) {
      throw new SecureLensValidationError('Vector database is required', {
        fieldErrors: { vectorDb: ['vectorDb must be specified'] },
      });
    }

    if (request.metadata && request.metadata.length !== request.documents.length) {
      throw new SecureLensValidationError('Metadata length mismatch', {
        fieldErrors: {
          metadata: ['Metadata array must have the same length as documents array'],
        },
      });
    }
  }

  private normalizeInterceptRequest(request: GateInterceptRequest): Record<string, unknown> {
    return {
      query: request.query,
      contextSource: this.normalizeVectorDb(request.contextSource),
      policies: request.policies,
      userContext: request.userContext,
      sessionId: request.sessionId,
      metadata: request.metadata,
    };
  }

  private normalizeQueryRequest(request: GateQueryRequest): Record<string, unknown> {
    return {
      query: request.query,
      contextSource: this.normalizeVectorDb(request.contextSource),
      llmProvider: this.normalizeLLM(request.llmProvider),
      model: request.model,
      topK: request.topK,
      policies: request.policies,
      userContext: request.userContext,
      sessionId: request.sessionId,
      systemPrompt: request.systemPrompt,
      stream: request.stream,
      metadata: request.metadata,
    };
  }

  private normalizeEmbedRequest(request: GateEmbedRequest): Record<string, unknown> {
    return {
      documents: request.documents,
      metadata: request.metadata,
      vectorDb: this.normalizeVectorDb(request.vectorDb),
      classifyBeforeEmbed: request.classifyBeforeEmbed,
      policies: request.policies,
      chunkSize: request.chunkSize,
      chunkOverlap: request.chunkOverlap,
    };
  }

  private normalizeVectorDb(
    source?: VectorDbProvider | VectorDbConfig
  ): VectorDbConfig | undefined {
    if (!source) return undefined;

    if (typeof source === 'string') {
      return { provider: source };
    }

    return source;
  }

  private normalizeLLM(provider?: LLMProvider | LLMConfig): LLMConfig | undefined {
    if (!provider) return undefined;

    if (typeof provider === 'string') {
      return { provider };
    }

    return provider;
  }

  private getApiKey(): string {
    // Access the API key through a method that would need to be exposed
    // For now, we'll use a workaround through the client's auth headers
    return (this.client as unknown as { config: { apiKey: string } }).config?.apiKey ?? '';
  }
}
