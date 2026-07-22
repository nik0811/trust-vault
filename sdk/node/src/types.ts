/**
 * SecureLens SDK - TypeScript Type Definitions
 *
 * Core types for the SecureLens AI Gate SDK.
 */

// ============================================================================
// Configuration Types
// ============================================================================

export interface SecureLensConfig {
  /** API key for authentication (format: sl_...) */
  apiKey: string;
  /** Base URL for the SecureLens API */
  baseUrl?: string;
  /** Tenant ID for multi-tenant deployments */
  tenantId?: string;
  /** Request timeout in milliseconds */
  timeout?: number;
  /** Number of retry attempts for failed requests */
  retries?: number;
  /** Enable debug logging */
  debug?: boolean;
}

export interface RequestOptions {
  /** Override timeout for this request */
  timeout?: number;
  /** Custom headers to include */
  headers?: Record<string, string>;
  /** Abort signal for request cancellation */
  signal?: AbortSignal;
}

// ============================================================================
// Vector Database Types
// ============================================================================

export type VectorDbProvider = 'qdrant' | 'pinecone' | 'weaviate' | 'milvus' | 'chroma' | 'custom';

export interface VectorDbConfig {
  provider: VectorDbProvider;
  /** Connection URL or endpoint */
  url?: string;
  /** API key for the vector database */
  apiKey?: string;
  /** Collection/index name */
  collection?: string;
  /** Custom configuration options */
  options?: Record<string, unknown>;
}

// ============================================================================
// LLM Provider Types
// ============================================================================

export type LLMProvider = 'openai' | 'anthropic' | 'azure' | 'bedrock' | 'cohere' | 'custom';

export interface LLMConfig {
  provider: LLMProvider;
  /** Model identifier (e.g., 'gpt-4', 'claude-3-opus') */
  model?: string;
  /** API key for the LLM provider */
  apiKey?: string;
  /** Custom endpoint URL */
  endpoint?: string;
  /** Temperature for generation */
  temperature?: number;
  /** Maximum tokens to generate */
  maxTokens?: number;
  /** Custom configuration options */
  options?: Record<string, unknown>;
}

// ============================================================================
// Classification Types
// ============================================================================

export type SensitivityLevel = 'public' | 'internal' | 'confidential' | 'restricted' | 'top_secret';

export type ClassificationType =
  | 'pii'
  | 'phi'
  | 'pci'
  | 'financial'
  | 'legal'
  | 'intellectual_property'
  | 'trade_secret'
  | 'custom';

export interface Classification {
  /** Type of classification */
  type: ClassificationType;
  /** Specific label (e.g., 'email', 'ssn', 'credit_card') */
  label: string;
  /** Confidence score (0-1) */
  confidence: number;
  /** Start position in text */
  start: number;
  /** End position in text */
  end: number;
  /** The matched text (may be redacted) */
  text?: string;
  /** Sensitivity level */
  sensitivity: SensitivityLevel;
}

export interface ClassificationResult {
  /** Original text (may be redacted based on policy) */
  text: string;
  /** List of classifications found */
  classifications: Classification[];
  /** Overall sensitivity level */
  overallSensitivity: SensitivityLevel;
  /** Processing time in milliseconds */
  processingTimeMs: number;
}

// ============================================================================
// AI Gate Request/Response Types
// ============================================================================

export interface GateInterceptRequest {
  /** The user query to intercept */
  query: string;
  /** Source of context documents */
  contextSource?: VectorDbProvider | VectorDbConfig;
  /** Policies to apply */
  policies?: string[];
  /** User context for policy evaluation */
  userContext?: UserContext;
  /** Session ID for conversation tracking */
  sessionId?: string;
  /** Additional metadata */
  metadata?: Record<string, unknown>;
}

export interface GateInterceptResponse {
  /** Whether the query is allowed */
  allowed: boolean;
  /** The safe (potentially modified) query */
  safeQuery: string;
  /** Original query (if different from safeQuery) */
  originalQuery: string;
  /** Classifications found in the query */
  classifications: Classification[];
  /** Policies that were applied */
  appliedPolicies: AppliedPolicy[];
  /** Audit trail ID */
  auditId: string;
  /** Reason if blocked */
  blockReason?: string;
  /** Suggested alternative query if blocked */
  suggestedQuery?: string;
}

export interface GateQueryRequest {
  /** The user query */
  query: string;
  /** Vector database for context retrieval */
  contextSource?: VectorDbProvider | VectorDbConfig;
  /** LLM provider configuration */
  llmProvider?: LLMProvider | LLMConfig;
  /** Model to use */
  model?: string;
  /** Number of context documents to retrieve */
  topK?: number;
  /** Policies to apply */
  policies?: string[];
  /** User context for policy evaluation */
  userContext?: UserContext;
  /** Session ID for conversation tracking */
  sessionId?: string;
  /** System prompt override */
  systemPrompt?: string;
  /** Enable streaming response */
  stream?: boolean;
  /** Additional metadata */
  metadata?: Record<string, unknown>;
}

export interface GateQueryResponse {
  /** The LLM response */
  response: string;
  /** The safe query that was sent */
  safeQuery: string;
  /** Classifications found in query and response */
  classifications: Classification[];
  /** Audit trail ID */
  auditId: string;
  /** Token usage statistics */
  tokensUsed: TokenUsage;
  /** Retrieved context documents */
  context?: ContextDocument[];
  /** Policies that were applied */
  appliedPolicies: AppliedPolicy[];
  /** Processing time in milliseconds */
  processingTimeMs: number;
}

export interface GateEmbedRequest {
  /** Documents to embed */
  documents: string[];
  /** Metadata for each document */
  metadata?: Record<string, unknown>[];
  /** Target vector database */
  vectorDb: VectorDbProvider | VectorDbConfig;
  /** Classify documents before embedding */
  classifyBeforeEmbed?: boolean;
  /** Policies to apply during embedding */
  policies?: string[];
  /** Chunk size for large documents */
  chunkSize?: number;
  /** Chunk overlap */
  chunkOverlap?: number;
}

export interface GateEmbedResponse {
  /** Number of documents embedded */
  documentsEmbedded: number;
  /** Number of chunks created */
  chunksCreated: number;
  /** Classifications found (if classifyBeforeEmbed was true) */
  classifications?: Classification[];
  /** Audit trail ID */
  auditId: string;
  /** Processing time in milliseconds */
  processingTimeMs: number;
  /** IDs of embedded documents in vector DB */
  documentIds: string[];
}

// ============================================================================
// Supporting Types
// ============================================================================

export interface UserContext {
  /** User identifier */
  userId?: string;
  /** User roles */
  roles?: string[];
  /** User groups */
  groups?: string[];
  /** Department */
  department?: string;
  /** Clearance level */
  clearanceLevel?: SensitivityLevel;
  /** Custom attributes */
  attributes?: Record<string, unknown>;
}

export interface AppliedPolicy {
  /** Policy identifier */
  policyId: string;
  /** Policy name */
  name: string;
  /** Action taken */
  action: 'allow' | 'block' | 'redact' | 'mask' | 'audit';
  /** Details about the action */
  details?: string;
}

export interface TokenUsage {
  /** Prompt tokens */
  promptTokens: number;
  /** Completion tokens */
  completionTokens: number;
  /** Total tokens */
  totalTokens: number;
  /** Estimated cost in USD */
  estimatedCost?: number;
}

export interface ContextDocument {
  /** Document ID */
  id: string;
  /** Document content */
  content: string;
  /** Similarity score */
  score: number;
  /** Document metadata */
  metadata?: Record<string, unknown>;
  /** Classifications in this document */
  classifications?: Classification[];
}

// ============================================================================
// Audit Types
// ============================================================================

export interface AuditEvent {
  /** Audit event ID */
  id: string;
  /** Timestamp */
  timestamp: string;
  /** Event type */
  eventType: 'query' | 'intercept' | 'embed' | 'classify';
  /** User context */
  userContext?: UserContext;
  /** Request summary */
  request: Record<string, unknown>;
  /** Response summary */
  response: Record<string, unknown>;
  /** Policies applied */
  policies: AppliedPolicy[];
  /** Classifications found */
  classifications: Classification[];
  /** Processing time */
  processingTimeMs: number;
}

export interface AuditQueryOptions {
  /** Start date */
  startDate?: Date;
  /** End date */
  endDate?: Date;
  /** User ID filter */
  userId?: string;
  /** Event type filter */
  eventType?: AuditEvent['eventType'];
  /** Policy ID filter */
  policyId?: string;
  /** Limit results */
  limit?: number;
  /** Offset for pagination */
  offset?: number;
}

// ============================================================================
// Streaming Types
// ============================================================================

export interface StreamChunk {
  /** Chunk type */
  type: 'content' | 'classification' | 'policy' | 'done' | 'error';
  /** Content for 'content' type */
  content?: string;
  /** Classification for 'classification' type */
  classification?: Classification;
  /** Policy for 'policy' type */
  policy?: AppliedPolicy;
  /** Final response for 'done' type */
  finalResponse?: GateQueryResponse;
  /** Error for 'error' type */
  error?: string;
}

export type StreamCallback = (chunk: StreamChunk) => void;

// ============================================================================
// Health & Status Types
// ============================================================================

export interface HealthStatus {
  /** Overall status */
  status: 'healthy' | 'degraded' | 'unhealthy';
  /** API version */
  version: string;
  /** Component statuses */
  components: {
    api: ComponentStatus;
    classifier: ComponentStatus;
    vectorDb?: ComponentStatus;
    llm?: ComponentStatus;
  };
  /** Server timestamp */
  timestamp: string;
}

export interface ComponentStatus {
  status: 'healthy' | 'degraded' | 'unhealthy';
  latencyMs?: number;
  message?: string;
}
