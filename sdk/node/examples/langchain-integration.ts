/**
 * LangChain.js Integration Example
 *
 * Demonstrates how to integrate SecureLens with LangChain.js
 * for governed RAG applications.
 *
 * Run: npx ts-node examples/langchain-integration.ts
 *
 * Note: This example requires langchain to be installed:
 *   npm install langchain @langchain/openai
 */

import { SecureLensClient, AIGate } from '../src';
import {
  SecureLensCallbackHandler,
  createSecureLensRetriever,
  type Document,
  type Retriever,
} from '../src/langchain';

// Initialize SecureLens
const client = new SecureLensClient({
  apiKey: process.env.SECURELENS_API_KEY ?? 'sl_test_key_here',
  baseUrl: process.env.SECURELENS_BASE_URL ?? 'https://api.securelens.ai',
  debug: true,
});

const gate = new AIGate(client);

// ============================================================================
// Example 1: Basic Callback Handler
// ============================================================================

async function basicCallbackExample() {
  console.log('\n--- Example 1: Basic Callback Handler ---\n');

  // Create the callback handler
  const secureLensHandler = new SecureLensCallbackHandler(gate, {
    policies: ['redact_pii', 'block_sensitive_queries'],
    blockOnPolicyViolation: true,
    classifyRetrievedDocs: true,
    classifyResponses: true,

    // Callbacks for monitoring
    onClassification: (classifications, source) => {
      console.log(`[SecureLens] Found ${classifications.length} classifications in ${source}:`);
      for (const c of classifications) {
        console.log(`  - ${c.type}/${c.label}: "${c.text}" (${c.confidence.toFixed(2)})`);
      }
    },

    onPolicyApplied: (policy) => {
      console.log(`[SecureLens] Policy applied: ${policy.name} (${policy.action})`);
    },

    onBlocked: (result) => {
      console.log(`[SecureLens] Query blocked: ${result.blockReason}`);
      if (result.suggestedQuery) {
        console.log(`[SecureLens] Suggested alternative: ${result.suggestedQuery}`);
      }
    },
  });

  // In a real application, you would use this with LangChain:
  //
  // import { ChatOpenAI } from '@langchain/openai';
  // import { RetrievalQAChain } from 'langchain/chains';
  //
  // const llm = new ChatOpenAI({
  //   modelName: 'gpt-4',
  //   callbacks: [secureLensHandler]
  // });
  //
  // const chain = RetrievalQAChain.fromLLM(llm, retriever, {
  //   callbacks: [secureLensHandler]
  // });
  //
  // const result = await chain.call({ query: "What is John's salary?" });

  // Simulate chain execution for demonstration
  console.log('Simulating LangChain execution with SecureLens handler...');

  // Simulate handleChainStart
  const runId = 'run_' + Date.now();
  try {
    await secureLensHandler.handleChainStart(
      { name: 'RetrievalQAChain' },
      { query: "What is John Smith's salary?" },
      runId
    );
    console.log('Chain started successfully');
  } catch (error) {
    console.log('Chain blocked:', (error as Error).message);
    return;
  }

  // Simulate handleRetrieverEnd
  await secureLensHandler.handleRetrieverEnd(
    [
      {
        pageContent: 'John Smith earns $150,000 as a Senior Engineer.',
        metadata: { source: 'hr_docs' },
      },
      {
        pageContent: 'The engineering team has 50 members.',
        metadata: { source: 'company_info' },
      },
    ],
    runId + '_retriever',
    runId
  );

  // Simulate handleLLMEnd
  await secureLensHandler.handleLLMEnd(
    {
      generations: [
        [{ text: "Based on the documents, John Smith's salary is $150,000." }],
      ],
    },
    runId + '_llm',
    runId
  );

  // Simulate handleChainEnd
  await secureLensHandler.handleChainEnd(
    { result: "John Smith's salary is $150,000 per year." },
    runId
  );

  console.log('\nChain execution completed');
}

// ============================================================================
// Example 2: Secure Retriever Wrapper
// ============================================================================

async function secureRetrieverExample() {
  console.log('\n--- Example 2: Secure Retriever Wrapper ---\n');

  // Create a mock base retriever (in real use, this would be your vector store retriever)
  const mockRetriever: Retriever = {
    async getRelevantDocuments(_query: string): Promise<Document[]> {
      return [
        {
          pageContent: 'John Smith (SSN: 123-45-6789) earns $150,000 annually.',
          metadata: { source: 'hr_confidential', department: 'hr' },
        },
        {
          pageContent: 'The company was founded in 2020 in San Francisco.',
          metadata: { source: 'company_info', department: 'general' },
        },
        {
          pageContent: 'Q3 revenue was $10M with projections of $15M for Q4.',
          metadata: { source: 'financial_reports', department: 'finance' },
        },
        {
          pageContent: 'Top secret project codename: Phoenix. Launch date: 2024.',
          metadata: { source: 'classified', department: 'r&d' },
        },
      ];
    },
  };

  // Wrap with SecureLens for classification and filtering
  const secureRetriever = createSecureLensRetriever(gate, mockRetriever, {
    classifyDocuments: true,
    filterSensitive: true,
    maxSensitivity: 'confidential', // Filter out 'restricted' and 'top_secret'
    userContext: {
      userId: 'user_123',
      roles: ['analyst'],
      clearanceLevel: 'confidential',
    },
  });

  console.log('Retrieving documents with sensitivity filtering...\n');

  const documents = await secureRetriever.getRelevantDocuments('company information');

  console.log(`Retrieved ${documents.length} documents (filtered by sensitivity):\n`);

  for (const doc of documents) {
    const secureLensMetadata = doc.metadata.secureLens as {
      sensitivity: string;
      classifications: Array<{ label: string }>;
    };

    console.log(`Source: ${doc.metadata.source}`);
    console.log(`Sensitivity: ${secureLensMetadata?.sensitivity ?? 'unknown'}`);
    console.log(`Content: ${doc.pageContent.slice(0, 80)}...`);
    console.log(
      `Classifications: ${secureLensMetadata?.classifications?.map((c) => c.label).join(', ') || 'none'}`
    );
    console.log('---');
  }
}

// ============================================================================
// Example 3: Full RAG Pipeline with SecureLens
// ============================================================================

async function fullRagPipelineExample() {
  console.log('\n--- Example 3: Full RAG Pipeline ---\n');

  // This example shows how you would structure a complete RAG pipeline
  // with SecureLens governance at every step

  const query = 'What are the salary ranges for engineers?';

  console.log(`Query: "${query}"\n`);

  // Step 1: Intercept and validate the query
  console.log('Step 1: Intercepting query...');
  const interceptResult = await gate.intercept({
    query,
    policies: ['redact_pii', 'salary_data_access'],
    userContext: {
      userId: 'user_123',
      roles: ['hr_manager'],
      department: 'hr',
      clearanceLevel: 'confidential',
    },
  });

  console.log(`  Allowed: ${interceptResult.allowed}`);
  console.log(`  Safe Query: ${interceptResult.safeQuery}`);
  console.log(`  Classifications: ${interceptResult.classifications.length}`);

  if (!interceptResult.allowed) {
    console.log(`  Blocked: ${interceptResult.blockReason}`);
    return;
  }

  // Step 2: Use the full gate.query for governed RAG
  console.log('\nStep 2: Executing governed query...');
  const queryResult = await gate.query({
    query: interceptResult.safeQuery,
    contextSource: 'qdrant',
    llmProvider: 'openai',
    model: 'gpt-4',
    topK: 5,
    policies: ['redact_pii', 'salary_data_access'],
    userContext: {
      userId: 'user_123',
      roles: ['hr_manager'],
      clearanceLevel: 'confidential',
    },
  });

  console.log(`  Response: ${queryResult.response.slice(0, 100)}...`);
  console.log(`  Tokens Used: ${queryResult.tokensUsed.totalTokens}`);
  console.log(`  Audit ID: ${queryResult.auditId}`);
  console.log(`  Applied Policies: ${queryResult.appliedPolicies.map((p) => p.name).join(', ')}`);

  // Step 3: Review audit trail
  console.log('\nStep 3: Reviewing audit trail...');
  const auditEvent = await gate.getAuditEvent(queryResult.auditId);
  console.log(`  Event Type: ${auditEvent.eventType}`);
  console.log(`  Timestamp: ${auditEvent.timestamp}`);
  console.log(`  Classifications Found: ${auditEvent.classifications.length}`);
}

// ============================================================================
// Example 4: Custom LangChain Chain with SecureLens
// ============================================================================

async function customChainExample() {
  console.log('\n--- Example 4: Custom Chain Pattern ---\n');

  // This shows a pattern for building custom chains with SecureLens

  interface ChainInput {
    query: string;
    userId: string;
  }

  interface ChainOutput {
    answer: string;
    sources: string[];
    auditId: string;
    wasRedacted: boolean;
  }

  // Custom chain class
  class SecureRAGChain {
    private gate: AIGate;
    private policies: string[];

    constructor(gate: AIGate, policies: string[] = []) {
      this.gate = gate;
      this.policies = policies;
    }

    async call(input: ChainInput): Promise<ChainOutput> {
      // Intercept
      const interceptResult = await this.gate.intercept({
        query: input.query,
        policies: this.policies,
        userContext: { userId: input.userId },
      });

      if (!interceptResult.allowed) {
        throw new Error(`Query blocked: ${interceptResult.blockReason}`);
      }

      // Query
      const queryResult = await this.gate.query({
        query: interceptResult.safeQuery,
        contextSource: 'qdrant',
        llmProvider: 'openai',
        model: 'gpt-4',
        topK: 3,
        policies: this.policies,
        userContext: { userId: input.userId },
      });

      return {
        answer: queryResult.response,
        sources: queryResult.context?.map((c) => c.id) ?? [],
        auditId: queryResult.auditId,
        wasRedacted: interceptResult.safeQuery !== input.query,
      };
    }
  }

  // Use the custom chain
  const chain = new SecureRAGChain(gate, ['redact_pii', 'financial_data_access']);

  try {
    const result = await chain.call({
      query: 'What is the company revenue?',
      userId: 'user_123',
    });

    console.log('Answer:', result.answer);
    console.log('Sources:', result.sources);
    console.log('Audit ID:', result.auditId);
    console.log('Was Redacted:', result.wasRedacted);
  } catch (error) {
    console.error('Chain error:', (error as Error).message);
  }
}

// ============================================================================
// Run all examples
// ============================================================================

async function main() {
  console.log('='.repeat(60));
  console.log('SecureLens + LangChain.js Integration Examples');
  console.log('='.repeat(60));

  try {
    await basicCallbackExample();
    await secureRetrieverExample();
    await fullRagPipelineExample();
    await customChainExample();
  } catch (error) {
    console.error('Example failed:', error);
  }

  console.log('\n' + '='.repeat(60));
  console.log('Examples completed');
  console.log('='.repeat(60));
}

main().catch(console.error);
