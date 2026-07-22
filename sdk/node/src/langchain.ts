/**
 * SecureLens SDK - LangChain.js Integration
 *
 * Callback handlers and utilities for integrating SecureLens with LangChain.js.
 */

import { AIGate } from './gate.js';
import type {
  Classification,
  GateInterceptResponse,
  UserContext,
  AppliedPolicy,
} from './types.js';
import { SecureLensPolicyError } from './errors.js';

/**
 * LangChain callback handler types (simplified for compatibility).
 * These match the LangChain.js callback interface.
 */
export interface LangChainCallbackHandlerMethods {
  handleLLMStart?: (
    llm: { name: string },
    prompts: string[],
    runId: string,
    parentRunId?: string,
    extraParams?: Record<string, unknown>
  ) => Promise<void>;

  handleLLMEnd?: (
    output: { generations: Array<Array<{ text: string }>> },
    runId: string,
    parentRunId?: string
  ) => Promise<void>;

  handleLLMError?: (error: Error, runId: string, parentRunId?: string) => Promise<void>;

  handleChainStart?: (
    chain: { name: string },
    inputs: Record<string, unknown>,
    runId: string,
    parentRunId?: string
  ) => Promise<void>;

  handleChainEnd?: (
    outputs: Record<string, unknown>,
    runId: string,
    parentRunId?: string
  ) => Promise<void>;

  handleChainError?: (error: Error, runId: string, parentRunId?: string) => Promise<void>;

  handleRetrieverStart?: (
    retriever: { name: string },
    query: string,
    runId: string,
    parentRunId?: string
  ) => Promise<void>;

  handleRetrieverEnd?: (
    documents: Array<{ pageContent: string; metadata: Record<string, unknown> }>,
    runId: string,
    parentRunId?: string
  ) => Promise<void>;

  handleRetrieverError?: (error: Error, runId: string, parentRunId?: string) => Promise<void>;
}

/**
 * Configuration for the SecureLens LangChain callback handler.
 */
export interface SecureLensCallbackOptions {
  /** Policies to apply */
  policies?: string[];
  /** User context for policy evaluation */
  userContext?: UserContext;
  /** Block queries that fail policy checks */
  blockOnPolicyViolation?: boolean;
  /** Classify retrieved documents */
  classifyRetrievedDocs?: boolean;
  /** Classify LLM responses */
  classifyResponses?: boolean;
  /** Called when classifications are found */
  onClassification?: (classifications: Classification[], source: string) => void;
  /** Called when a policy is applied */
  onPolicyApplied?: (policy: AppliedPolicy) => void;
  /** Called when a query is blocked */
  onBlocked?: (result: GateInterceptResponse) => void;
}

/**
 * Run context for tracking state across callbacks.
 */
interface RunContext {
  query?: string;
  interceptResult?: GateInterceptResponse;
  classifications: Classification[];
  appliedPolicies: AppliedPolicy[];
  startTime: number;
}

/**
 * SecureLens callback handler for LangChain.js.
 *
 * Integrates SecureLens AI Gate with LangChain chains and agents
 * to provide governance, classification, and audit capabilities.
 *
 * @example
 * ```typescript
 * import { ChatOpenAI } from 'langchain/chat_models/openai';
 * import { RetrievalQAChain } from 'langchain/chains';
 * import { SecureLensClient, AIGate } from '@securelens/sdk';
 * import { SecureLensCallbackHandler } from '@securelens/sdk/langchain';
 *
 * const client = new SecureLensClient({ apiKey: 'sl_...' });
 * const gate = new AIGate(client);
 *
 * const handler = new SecureLensCallbackHandler(gate, {
 *   policies: ['redact_pii', 'block_sensitive'],
 *   blockOnPolicyViolation: true,
 *   classifyRetrievedDocs: true,
 *   onClassification: (classifications, source) => {
 *     console.log(`Found ${classifications.length} classifications in ${source}`);
 *   }
 * });
 *
 * const chain = RetrievalQAChain.fromLLM(llm, retriever, {
 *   callbacks: [handler]
 * });
 *
 * const result = await chain.call({ query: "What is John's salary?" });
 * ```
 */
export class SecureLensCallbackHandler implements LangChainCallbackHandlerMethods {
  readonly name = 'SecureLensCallbackHandler';

  private readonly gate: AIGate;
  private readonly options: SecureLensCallbackOptions;
  private readonly runContexts: Map<string, RunContext> = new Map();

  constructor(gate: AIGate, options: SecureLensCallbackOptions = {}) {
    this.gate = gate;
    this.options = {
      blockOnPolicyViolation: true,
      classifyRetrievedDocs: true,
      classifyResponses: true,
      ...options,
    };
  }

  /**
   * Handle chain start - intercept the query.
   */
  async handleChainStart(
    _chain: { name: string },
    inputs: Record<string, unknown>,
    runId: string,
    _parentRunId?: string
  ): Promise<void> {
    const query = this.extractQuery(inputs);
    if (!query) return;

    const context: RunContext = {
      query,
      classifications: [],
      appliedPolicies: [],
      startTime: Date.now(),
    };

    try {
      const result = await this.gate.intercept({
        query,
        policies: this.options.policies,
        userContext: this.options.userContext,
      });

      context.interceptResult = result;
      context.classifications.push(...result.classifications);
      context.appliedPolicies.push(...result.appliedPolicies);

      if (result.classifications.length > 0) {
        this.options.onClassification?.(result.classifications, 'query');
      }

      for (const policy of result.appliedPolicies) {
        this.options.onPolicyApplied?.(policy);
      }

      if (!result.allowed && this.options.blockOnPolicyViolation) {
        this.options.onBlocked?.(result);
        throw new SecureLensPolicyError(result.blockReason ?? 'Query blocked by policy', {
          suggestedQuery: result.suggestedQuery,
        });
      }
    } catch (error) {
      if (error instanceof SecureLensPolicyError) {
        throw error;
      }
      // Log but don't block on intercept errors
      console.warn('[SecureLens] Failed to intercept query:', error);
    }

    this.runContexts.set(runId, context);
  }

  /**
   * Handle chain end - audit the result.
   */
  async handleChainEnd(
    outputs: Record<string, unknown>,
    runId: string,
    _parentRunId?: string
  ): Promise<void> {
    const context = this.runContexts.get(runId);
    if (!context) return;

    try {
      if (this.options.classifyResponses) {
        const responseText = this.extractResponse(outputs);
        if (responseText) {
          const result = await this.gate.classify(responseText);
          context.classifications.push(...result.classifications);

          if (result.classifications.length > 0) {
            this.options.onClassification?.(result.classifications, 'response');
          }
        }
      }
    } catch (error) {
      console.warn('[SecureLens] Failed to classify response:', error);
    } finally {
      this.runContexts.delete(runId);
    }
  }

  /**
   * Handle chain error.
   */
  async handleChainError(
    _error: Error,
    runId: string,
    _parentRunId?: string
  ): Promise<void> {
    this.runContexts.delete(runId);
  }

  /**
   * Handle retriever start.
   */
  async handleRetrieverStart(
    _retriever: { name: string },
    query: string,
    runId: string,
    parentRunId?: string
  ): Promise<void> {
    // Get or create context
    let context = parentRunId ? this.runContexts.get(parentRunId) : undefined;
    if (!context) {
      context = {
        query,
        classifications: [],
        appliedPolicies: [],
        startTime: Date.now(),
      };
      this.runContexts.set(runId, context);
    }
  }

  /**
   * Handle retriever end - classify retrieved documents.
   */
  async handleRetrieverEnd(
    documents: Array<{ pageContent: string; metadata: Record<string, unknown> }>,
    runId: string,
    parentRunId?: string
  ): Promise<void> {
    if (!this.options.classifyRetrievedDocs || documents.length === 0) return;

    const context = this.runContexts.get(parentRunId ?? runId);
    if (!context) return;

    try {
      const texts = documents.map((doc) => doc.pageContent);
      const results = await this.gate.classifyBatch(texts);

      for (const result of results) {
        context.classifications.push(...result.classifications);
      }

      const allClassifications = results.flatMap((r) => r.classifications);
      if (allClassifications.length > 0) {
        this.options.onClassification?.(allClassifications, 'retrieved_docs');
      }
    } catch (error) {
      console.warn('[SecureLens] Failed to classify retrieved documents:', error);
    }
  }

  /**
   * Handle retriever error.
   */
  async handleRetrieverError(
    _error: Error,
    runId: string,
    _parentRunId?: string
  ): Promise<void> {
    this.runContexts.delete(runId);
  }

  /**
   * Handle LLM start.
   */
  async handleLLMStart(
    _llm: { name: string },
    _prompts: string[],
    _runId: string,
    _parentRunId?: string,
    _extraParams?: Record<string, unknown>
  ): Promise<void> {
    // The prompt has already been intercepted at chain level
  }

  /**
   * Handle LLM end - classify response.
   */
  async handleLLMEnd(
    output: { generations: Array<Array<{ text: string }>> },
    runId: string,
    parentRunId?: string
  ): Promise<void> {
    if (!this.options.classifyResponses) return;

    const context = this.runContexts.get(parentRunId ?? runId);
    if (!context) return;

    try {
      const texts = output.generations.flatMap((gen) => gen.map((g) => g.text));
      if (texts.length === 0) return;

      const results = await this.gate.classifyBatch(texts);

      for (const result of results) {
        context.classifications.push(...result.classifications);
      }

      const allClassifications = results.flatMap((r) => r.classifications);
      if (allClassifications.length > 0) {
        this.options.onClassification?.(allClassifications, 'llm_response');
      }
    } catch (error) {
      console.warn('[SecureLens] Failed to classify LLM response:', error);
    }
  }

  /**
   * Handle LLM error.
   */
  async handleLLMError(
    _error: Error,
    _runId: string,
    _parentRunId?: string
  ): Promise<void> {
    // No cleanup needed
  }

  /**
   * Get classifications collected during a run.
   */
  getClassifications(runId: string): Classification[] {
    return this.runContexts.get(runId)?.classifications ?? [];
  }

  /**
   * Get applied policies for a run.
   */
  getAppliedPolicies(runId: string): AppliedPolicy[] {
    return this.runContexts.get(runId)?.appliedPolicies ?? [];
  }

  /**
   * Extract query from chain inputs.
   */
  private extractQuery(inputs: Record<string, unknown>): string | undefined {
    return (
      (inputs.query as string) ??
      (inputs.question as string) ??
      (inputs.input as string) ??
      (inputs.prompt as string)
    );
  }

  /**
   * Extract response from chain outputs.
   */
  private extractResponse(outputs: Record<string, unknown>): string | undefined {
    return (
      (outputs.result as string) ??
      (outputs.answer as string) ??
      (outputs.output as string) ??
      (outputs.response as string) ??
      (outputs.text as string)
    );
  }
}

/**
 * Create a SecureLens-wrapped retriever.
 *
 * Wraps any LangChain retriever to add classification and governance.
 *
 * @example
 * ```typescript
 * import { createSecureLensRetriever } from '@securelens/sdk/langchain';
 *
 * const secureRetriever = createSecureLensRetriever(gate, baseRetriever, {
 *   classifyDocuments: true,
 *   filterSensitive: true,
 *   maxSensitivity: 'confidential'
 * });
 * ```
 */
export interface SecureLensRetrieverOptions {
  /** Classify retrieved documents */
  classifyDocuments?: boolean;
  /** Filter out documents above sensitivity threshold */
  filterSensitive?: boolean;
  /** Maximum allowed sensitivity level */
  maxSensitivity?: 'public' | 'internal' | 'confidential' | 'restricted';
  /** User context for filtering */
  userContext?: UserContext;
}

/**
 * Document interface matching LangChain's Document type.
 */
export interface Document {
  pageContent: string;
  metadata: Record<string, unknown>;
}

/**
 * Retriever interface matching LangChain's BaseRetriever.
 */
export interface Retriever {
  getRelevantDocuments(query: string): Promise<Document[]>;
}

export function createSecureLensRetriever(
  gate: AIGate,
  baseRetriever: Retriever,
  options: SecureLensRetrieverOptions = {}
): Retriever {
  const sensitivityOrder = ['public', 'internal', 'confidential', 'restricted', 'top_secret'];

  return {
    async getRelevantDocuments(query: string): Promise<Document[]> {
      const documents = await baseRetriever.getRelevantDocuments(query);

      if (!options.classifyDocuments && !options.filterSensitive) {
        return documents;
      }

      // Classify documents
      const texts = documents.map((doc) => doc.pageContent);
      const results = await gate.classifyBatch(texts);

      // Enrich documents with classifications
      const enrichedDocs = documents.map((doc, i) => ({
        ...doc,
        metadata: {
          ...doc.metadata,
          secureLens: {
            classifications: results[i]?.classifications ?? [],
            sensitivity: results[i]?.overallSensitivity ?? 'public',
          },
        },
      }));

      // Filter if needed
      if (options.filterSensitive && options.maxSensitivity) {
        const maxIndex = sensitivityOrder.indexOf(options.maxSensitivity);

        return enrichedDocs.filter((doc) => {
          const docSensitivity =
            (doc.metadata.secureLens as { sensitivity: string })?.sensitivity ?? 'public';
          const docIndex = sensitivityOrder.indexOf(docSensitivity);
          return docIndex <= maxIndex;
        });
      }

      return enrichedDocs;
    },
  };
}
