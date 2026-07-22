"""RAG integration example for SecureLens AI Gate."""

import asyncio
from securelens import SecureLensClient, AIGate


def sync_rag_example():
    """Synchronous RAG integration example."""
    print("=" * 60)
    print("Synchronous RAG Integration Example")
    print("=" * 60)

    # Initialize client and gate
    client = SecureLensClient(
        api_key="sl_your_api_key_here",
        base_url="https://api.securelens.ai"
    )
    gate = AIGate(client)

    # Step 1: Intercept the query first
    print("\n1. Intercepting query...")
    intercept_result = gate.intercept(
        query="What is the salary range for senior engineers?",
        context_source="qdrant",
        policies=["redact_pii", "audit_financial"]
    )

    print(f"   Original: {intercept_result.original_query}")
    print(f"   Safe: {intercept_result.safe_query}")
    print(f"   Classifications found: {len(intercept_result.classifications)}")

    # Step 2: Full RAG query with governance
    print("\n2. Executing governed RAG query...")
    query_result = gate.query(
        query="Summarize the Q4 financial projections",
        llm_provider="openai",
        model="gpt-4",
        context_source="qdrant",
        top_k=5,
        policies=["redact_financial_pii", "audit_all"],
        temperature=0.7,
        max_tokens=500,
        metadata={
            "user_id": "user_123",
            "session_id": "session_456",
            "department": "finance"
        }
    )

    print(f"   Response: {query_result.response[:200]}...")
    print(f"   Context chunks retrieved: {len(query_result.context_chunks)}")
    print(f"   Query classifications: {len(query_result.query_classifications)}")
    print(f"   Response classifications: {len(query_result.response_classifications)}")
    print(f"   Tokens used: {query_result.tokens_used}")
    print(f"   Processing time: {query_result.processing_time_ms}ms")
    print(f"   Audit ID: {query_result.audit_id}")

    # Step 3: Embed documents with governance
    print("\n3. Embedding documents with governance...")
    embed_result = gate.embed(
        documents=[
            "John Smith, Senior Engineer, earns $185,000 annually with a 15% bonus.",
            "Q4 revenue projections show 23% YoY growth with $45M in new contracts.",
            "The engineering team expanded to 150 members across 3 offices.",
        ],
        metadata=[
            {"source": "hr_system", "department": "engineering", "confidential": True},
            {"source": "finance_reports", "quarter": "Q4", "year": 2024},
            {"source": "company_updates", "public": True},
        ],
        vector_db="qdrant",
        collection="company_docs",
        classify_before_embed=True,
        policies=["redact_salary", "redact_pii"]
    )

    print(f"   Documents processed: {embed_result.documents_processed}")
    print(f"   Documents blocked: {embed_result.documents_blocked}")
    print(f"   Classifications found: {embed_result.classifications_found}")
    print(f"   Embeddings stored: {embed_result.embeddings_stored}")
    print(f"   Processing time: {embed_result.processing_time_ms}ms")

    if embed_result.blocked_documents:
        print("   Blocked documents:")
        for blocked in embed_result.blocked_documents:
            print(f"     - Index {blocked.get('index')}: {blocked.get('reason')}")

    client.close()


async def async_rag_example():
    """Asynchronous RAG integration example."""
    print("\n" + "=" * 60)
    print("Asynchronous RAG Integration Example")
    print("=" * 60)

    async with SecureLensClient(
        api_key="sl_your_api_key_here",
        base_url="https://api.securelens.ai"
    ) as client:
        gate = AIGate(client)

        # Parallel query interception
        print("\n1. Parallel query interception...")
        queries = [
            "What is John's email address?",
            "Show me the budget for Project Alpha",
            "List all employees in the engineering team",
        ]

        results = await asyncio.gather(*[
            gate.aintercept(query=q, policies=["redact_pii"])
            for q in queries
        ])

        for i, result in enumerate(results):
            print(f"   Query {i + 1}:")
            print(f"     Original: {result.original_query}")
            print(f"     Safe: {result.safe_query}")
            print(f"     Classifications: {len(result.classifications)}")

        # Async RAG query
        print("\n2. Async RAG query...")
        response = await gate.aquery(
            query="What are the key metrics for this quarter?",
            llm_provider="anthropic",
            model="claude-3-opus",
            context_source="pinecone",
            top_k=10,
            policies=["redact_financial_pii"]
        )

        print(f"   Response length: {len(response.response)} chars")
        print(f"   Audit ID: {response.audit_id}")


def main():
    """Run all examples."""
    # Run sync example
    sync_rag_example()

    # Run async example
    asyncio.run(async_rag_example())

    print("\n" + "=" * 60)
    print("Examples completed!")
    print("=" * 60)


if __name__ == "__main__":
    main()
