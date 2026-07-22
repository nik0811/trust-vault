# SecureLens Python SDK

Official Python SDK for [SecureLens AI Gate](https://securelens.ai) - Add data governance, classification, and policy enforcement to your RAG applications.

## Features

- **Query Interception**: Classify and redact sensitive data in queries before they reach your LLM
- **Full RAG Pipeline**: Governed query → context retrieval → LLM response with audit trail
- **Document Embedding**: Classify documents before embedding with policy enforcement
- **LangChain Integration**: Drop-in callback handler for existing LangChain applications
- **Async Support**: Full async/await support for high-performance applications
- **Type Safety**: Complete type hints and Pydantic models

## Installation

```bash
pip install securelens
```

With LangChain integration:

```bash
pip install securelens[langchain]
```

## Quick Start

### Initialize the Client

```python
from securelens import SecureLensClient, AIGate

# Initialize client
client = SecureLensClient(
    api_key="sl_your_api_key",
    base_url="https://api.securelens.ai"  # or your self-hosted URL
)

# Create AI Gate
gate = AIGate(client)
```

### Intercept and Classify Queries

```python
# Intercept a query to detect and redact sensitive data
result = gate.intercept(
    query="What is John Smith's salary and SSN?",
    policies=["redact_pii", "block_phi"]
)

print(f"Original: {result.original_query}")
print(f"Safe: {result.safe_query}")  # "What is [PERSON]'s salary and [SSN]?"
print(f"Classifications: {result.classifications}")
print(f"Audit ID: {result.audit_id}")

# Check if blocked by policy
if result.blocked:
    print(f"Blocked: {result.block_reason}")
```

### Full RAG Query

```python
# Execute a governed RAG query
response = gate.query(
    query="What are our Q4 revenue projections?",
    llm_provider="openai",
    model="gpt-4",
    context_source="qdrant",  # Your vector DB
    top_k=5,
    policies=["redact_financial_pii"]
)

print(f"Response: {response.response}")
print(f"Context chunks: {len(response.context_chunks)}")
print(f"Tokens used: {response.tokens_used}")
```

### Embed Documents with Governance

```python
# Embed documents with classification and policy enforcement
result = gate.embed(
    documents=[
        "John Smith earns $150,000 annually...",
        "Company confidential: Q4 projections show..."
    ],
    metadata=[
        {"source": "hr_docs", "department": "engineering"},
        {"source": "finance", "classification": "confidential"}
    ],
    vector_db="qdrant",
    classify_before_embed=True,
    policies=["redact_salary", "block_confidential"]
)

print(f"Embedded: {result.embeddings_stored}")
print(f"Blocked: {result.documents_blocked}")
```

## LangChain Integration

### Callback Handler

```python
from securelens import SecureLensClient, AIGate
from securelens.integrations import SecureLensLangChainCallback
from langchain.chains import RetrievalQA
from langchain_openai import ChatOpenAI

# Setup SecureLens
client = SecureLensClient(api_key="sl_...")
gate = AIGate(client)
callback = SecureLensLangChainCallback(
    gate=gate,
    policies=["redact_pii"],
    block_on_violation=False
)

# Use with LangChain
llm = ChatOpenAI(model="gpt-4")
chain = RetrievalQA.from_chain_type(
    llm=llm,
    retriever=your_retriever,
    callbacks=[callback]
)

# Queries are automatically governed
result = chain.invoke({"query": "What is John's salary?"})

# Access audit information
print(f"Audit IDs: {callback.audit_ids}")
```

### Secure Retriever Wrapper

```python
from securelens.integrations import SecureLensRetriever

# Wrap your existing retriever
secure_retriever = SecureLensRetriever(
    retriever=base_retriever,
    gate=gate,
    policies=["redact_pii"],
    redact_content=True
)

# Retrieved documents are automatically classified and redacted
docs = secure_retriever.get_relevant_documents("Find John Smith's records")
```

## Async Support

All methods have async variants:

```python
import asyncio
from securelens import SecureLensClient, AIGate

async def main():
    async with SecureLensClient(api_key="sl_...") as client:
        gate = AIGate(client)
        
        # Async intercept
        result = await gate.aintercept(
            query="What is John's SSN?",
            policies=["redact_pii"]
        )
        
        # Async query
        response = await gate.aquery(
            query="Summarize Q4 results",
            llm_provider="openai",
            model="gpt-4"
        )

asyncio.run(main())
```

## Error Handling

```python
from securelens import SecureLensClient, AIGate
from securelens.exceptions import (
    SecureLensAuthError,
    SecureLensRateLimitError,
    SecureLensPolicyError,
    SecureLensConnectionError,
)

client = SecureLensClient(api_key="sl_...")
gate = AIGate(client)

try:
    result = gate.intercept(query="sensitive query")
except SecureLensAuthError:
    print("Invalid or expired API key")
except SecureLensRateLimitError as e:
    print(f"Rate limited. Retry after {e.retry_after} seconds")
except SecureLensPolicyError as e:
    print(f"Blocked by policy: {e.policy_name}")
    print(f"Violations: {e.violations}")
except SecureLensConnectionError:
    print("Network connection failed")
```

## Configuration

### Multi-Tenant Setup

```python
client = SecureLensClient(
    api_key="sl_...",
    tenant_id="tenant_123",  # For multi-tenant deployments
)
```

### Custom Headers

```python
client = SecureLensClient(
    api_key="sl_...",
    custom_headers={
        "X-Request-ID": "req_123",
        "X-Correlation-ID": "corr_456"
    }
)
```

### Self-Hosted Deployment

```python
client = SecureLensClient(
    api_key="sl_...",
    base_url="https://securelens.internal.company.com",
    verify_ssl=True,  # Set to False for self-signed certs (not recommended)
)
```

## API Reference

### SecureLensClient

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `api_key` | str | required | API key (starts with `sl_`) |
| `base_url` | str | `https://api.securelens.ai` | API base URL |
| `tenant_id` | str | None | Tenant ID for multi-tenant |
| `timeout` | float | 30.0 | Request timeout in seconds |
| `verify_ssl` | bool | True | Verify SSL certificates |

### AIGate.intercept()

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `query` | str | required | Query to intercept |
| `context_source` | str | None | Vector DB source |
| `policies` | List[str] | [] | Policies to apply |
| `metadata` | Dict | {} | Additional metadata |

### AIGate.query()

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `query` | str | required | User query |
| `llm_provider` | str | required | LLM provider |
| `model` | str | required | Model name |
| `context_source` | str | None | Vector DB source |
| `top_k` | int | 5 | Context chunks to retrieve |
| `policies` | List[str] | [] | Policies to apply |
| `temperature` | float | 0.7 | LLM temperature |
| `max_tokens` | int | None | Max response tokens |

### AIGate.embed()

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `documents` | List[str] | required | Documents to embed |
| `vector_db` | str | required | Target vector DB |
| `metadata` | List[Dict] | [] | Document metadata |
| `classify_before_embed` | bool | True | Classify first |
| `policies` | List[str] | [] | Policies to apply |

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- Documentation: https://docs.securelens.ai
- Issues: https://github.com/securelens/securelens-python/issues
- Email: support@securelens.ai
