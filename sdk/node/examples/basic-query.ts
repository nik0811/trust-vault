/**
 * Basic Query Example
 *
 * Demonstrates how to use the SecureLens SDK to intercept and query
 * through the AI Gate with governance policies.
 *
 * Run: npx ts-node examples/basic-query.ts
 */

import { SecureLensClient, AIGate, isSecureLensError, SecureLensPolicyError } from '../src';

async function main() {
  // Initialize the client
  const client = new SecureLensClient({
    apiKey: process.env.SECURELENS_API_KEY ?? 'sl_test_key_here',
    baseUrl: process.env.SECURELENS_BASE_URL ?? 'https://api.securelens.ai',
    debug: true,
  });

  // Create the AI Gate
  const gate = new AIGate(client);

  // Check API health
  console.log('Checking API health...');
  try {
    const health = await client.health();
    console.log('API Status:', health.status);
    console.log('Version:', health.version);
  } catch (error) {
    console.error('Health check failed:', error);
  }

  // Example 1: Intercept a query
  console.log('\n--- Example 1: Intercept Query ---');
  try {
    const interceptResult = await gate.intercept({
      query: "What is John Smith's salary and SSN?",
      policies: ['redact_pii', 'block_salary_queries'],
      userContext: {
        userId: 'user_123',
        roles: ['analyst'],
        department: 'finance',
      },
    });

    console.log('Allowed:', interceptResult.allowed);
    console.log('Safe Query:', interceptResult.safeQuery);
    console.log('Classifications:', interceptResult.classifications.length);
    console.log('Audit ID:', interceptResult.auditId);

    if (!interceptResult.allowed) {
      console.log('Block Reason:', interceptResult.blockReason);
      console.log('Suggested Query:', interceptResult.suggestedQuery);
    }
  } catch (error) {
    if (error instanceof SecureLensPolicyError) {
      console.log('Query blocked by policy:', error.message);
      console.log('Policy:', error.policyName);
    } else {
      throw error;
    }
  }

  // Example 2: Full governed query
  console.log('\n--- Example 2: Full Query ---');
  try {
    const queryResult = await gate.query({
      query: 'What are the Q3 revenue projections?',
      contextSource: 'qdrant',
      llmProvider: 'openai',
      model: 'gpt-4',
      topK: 5,
      policies: ['financial_data_access'],
      userContext: {
        userId: 'user_123',
        roles: ['analyst'],
        clearanceLevel: 'confidential',
      },
    });

    console.log('Response:', queryResult.response);
    console.log('Safe Query:', queryResult.safeQuery);
    console.log('Tokens Used:', queryResult.tokensUsed.totalTokens);
    console.log('Audit ID:', queryResult.auditId);
    console.log('Processing Time:', queryResult.processingTimeMs, 'ms');

    if (queryResult.context) {
      console.log('Context Documents:', queryResult.context.length);
    }
  } catch (error) {
    if (isSecureLensError(error)) {
      console.error('SecureLens Error:', error.code, error.message);
    } else {
      throw error;
    }
  }

  // Example 3: Classify text
  console.log('\n--- Example 3: Classify Text ---');
  try {
    const classifyResult = await gate.classify(
      'Contact John Smith at john.smith@example.com or call 555-123-4567. ' +
        'His SSN is 123-45-6789 and credit card is 4111-1111-1111-1111.'
    );

    console.log('Overall Sensitivity:', classifyResult.overallSensitivity);
    console.log('Classifications found:');
    for (const classification of classifyResult.classifications) {
      console.log(
        `  - ${classification.type}/${classification.label}: ` +
          `"${classification.text}" (confidence: ${classification.confidence.toFixed(2)})`
      );
    }
  } catch (error) {
    console.error('Classification failed:', error);
  }

  // Example 4: Embed documents
  console.log('\n--- Example 4: Embed Documents ---');
  try {
    const embedResult = await gate.embed({
      documents: [
        'John Smith earns $150,000 annually as a Senior Engineer.',
        'The company was founded in 2020 and is headquartered in San Francisco.',
        'Our Q3 revenue was $10M with a 20% growth rate.',
      ],
      metadata: [
        { source: 'hr_docs', department: 'hr', confidential: true },
        { source: 'company_info', department: 'general', confidential: false },
        { source: 'financial_reports', department: 'finance', confidential: true },
      ],
      vectorDb: 'qdrant',
      classifyBeforeEmbed: true,
      chunkSize: 512,
      chunkOverlap: 50,
    });

    console.log('Documents Embedded:', embedResult.documentsEmbedded);
    console.log('Chunks Created:', embedResult.chunksCreated);
    console.log('Document IDs:', embedResult.documentIds);
    console.log('Audit ID:', embedResult.auditId);

    if (embedResult.classifications) {
      console.log('Classifications found:', embedResult.classifications.length);
    }
  } catch (error) {
    console.error('Embedding failed:', error);
  }

  // Example 5: Streaming query
  console.log('\n--- Example 5: Streaming Query ---');
  try {
    console.log('Response: ');
    const streamResult = await gate.queryStream(
      {
        query: 'Summarize the quarterly report in 3 bullet points.',
        llmProvider: 'openai',
        model: 'gpt-4',
        contextSource: 'qdrant',
        stream: true,
      },
      (chunk) => {
        switch (chunk.type) {
          case 'content':
            process.stdout.write(chunk.content ?? '');
            break;
          case 'classification':
            console.log('\n[Classification found]:', chunk.classification?.label);
            break;
          case 'policy':
            console.log('\n[Policy applied]:', chunk.policy?.name);
            break;
          case 'done':
            console.log('\n\nStream complete. Audit ID:', chunk.finalResponse?.auditId);
            break;
          case 'error':
            console.error('\nStream error:', chunk.error);
            break;
        }
      }
    );

    console.log('Total tokens:', streamResult.tokensUsed.totalTokens);
  } catch (error) {
    console.error('Streaming failed:', error);
  }

  // Example 6: Get audit events
  console.log('\n--- Example 6: Audit Events ---');
  try {
    const auditEvents = await gate.getAuditEvents({
      limit: 5,
      eventType: 'query',
    });

    console.log('Recent audit events:');
    for (const event of auditEvents) {
      console.log(`  - ${event.id}: ${event.eventType} at ${event.timestamp}`);
    }
  } catch (error) {
    console.error('Failed to get audit events:', error);
  }
}

main().catch(console.error);
