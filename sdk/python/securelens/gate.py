"""AI Gate for query interception and governance."""

from typing import Optional, List, Dict, Any

from securelens.client import SecureLensClient
from securelens.models import (
    InterceptResult,
    QueryResult,
    EmbedResult,
    Classification,
    PolicyViolation,
    ContextChunk,
)


class AIGate:
    """AI Gate for intercepting and governing queries to RAG systems.

    The AIGate provides three main capabilities:
    1. intercept() - Classify and optionally redact queries before processing
    2. query() - Full RAG query with context retrieval and LLM response
    3. embed() - Embed documents with classification and policy enforcement

    Example:
        >>> from securelens import SecureLensClient, AIGate
        >>> client = SecureLensClient(api_key="sl_...")
        >>> gate = AIGate(client)
        >>>
        >>> # Intercept a query
        >>> result = gate.intercept("What is John Smith's salary?")
        >>> print(result.safe_query)  # Redacted version
        >>> print(result.classifications)  # Detected PII
    """

    def __init__(self, client: SecureLensClient):
        """Initialize AI Gate.

        Args:
            client: SecureLensClient instance
        """
        self.client = client

    def intercept(
        self,
        query: str,
        context_source: Optional[str] = None,
        policies: Optional[List[str]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> InterceptResult:
        """Intercept and classify a query.

        Analyzes the query for sensitive data, applies policies, and returns
        a safe version with redactions if needed.

        Args:
            query: The query to intercept
            context_source: Optional vector DB source identifier
            policies: List of policy names to apply (e.g., ["redact_pii", "block_phi"])
            metadata: Additional metadata for audit logging

        Returns:
            InterceptResult with classifications and safe query

        Example:
            >>> result = gate.intercept(
            ...     query="What is John Smith's SSN?",
            ...     policies=["redact_pii"]
            ... )
            >>> print(result.safe_query)
            "What is [PERSON]'s [SSN]?"
        """
        data = {
            "query": query,
            "context_source": context_source,
            "policies": policies or [],
            "metadata": metadata or {},
        }

        response = self.client.post("/gate/intercept", data=data)
        return self._parse_intercept_result(response)

    async def aintercept(
        self,
        query: str,
        context_source: Optional[str] = None,
        policies: Optional[List[str]] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> InterceptResult:
        """Async version of intercept."""
        data = {
            "query": query,
            "context_source": context_source,
            "policies": policies or [],
            "metadata": metadata or {},
        }

        response = await self.client.apost("/gate/intercept", data=data)
        return self._parse_intercept_result(response)

    def query(
        self,
        query: str,
        llm_provider: str,
        model: str,
        context_source: Optional[str] = None,
        top_k: int = 5,
        policies: Optional[List[str]] = None,
        system_prompt: Optional[str] = None,
        temperature: float = 0.7,
        max_tokens: Optional[int] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> QueryResult:
        """Execute a full RAG query through the AI Gate.

        This method:
        1. Classifies the query for sensitive data
        2. Retrieves context from the vector DB
        3. Applies governance policies
        4. Sends to the LLM
        5. Classifies the response
        6. Returns the governed response

        Args:
            query: The user query
            llm_provider: LLM provider (openai, anthropic, azure, etc.)
            model: Model name (gpt-4, claude-3-opus, etc.)
            context_source: Vector DB source identifier
            top_k: Number of context chunks to retrieve
            policies: Policies to apply
            system_prompt: Custom system prompt
            temperature: LLM temperature
            max_tokens: Maximum response tokens
            metadata: Additional metadata

        Returns:
            QueryResult with response and audit information

        Example:
            >>> result = gate.query(
            ...     query="What are our Q4 revenue projections?",
            ...     llm_provider="openai",
            ...     model="gpt-4",
            ...     context_source="qdrant",
            ...     top_k=5
            ... )
            >>> print(result.response)
        """
        data = {
            "query": query,
            "llm_provider": llm_provider,
            "model": model,
            "context_source": context_source,
            "top_k": top_k,
            "policies": policies or [],
            "system_prompt": system_prompt,
            "temperature": temperature,
            "max_tokens": max_tokens,
            "metadata": metadata or {},
        }

        response = self.client.post("/gate/query", data=data)
        return self._parse_query_result(response)

    async def aquery(
        self,
        query: str,
        llm_provider: str,
        model: str,
        context_source: Optional[str] = None,
        top_k: int = 5,
        policies: Optional[List[str]] = None,
        system_prompt: Optional[str] = None,
        temperature: float = 0.7,
        max_tokens: Optional[int] = None,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> QueryResult:
        """Async version of query."""
        data = {
            "query": query,
            "llm_provider": llm_provider,
            "model": model,
            "context_source": context_source,
            "top_k": top_k,
            "policies": policies or [],
            "system_prompt": system_prompt,
            "temperature": temperature,
            "max_tokens": max_tokens,
            "metadata": metadata or {},
        }

        response = await self.client.apost("/gate/query", data=data)
        return self._parse_query_result(response)

    def embed(
        self,
        documents: List[str],
        vector_db: str,
        metadata: Optional[List[Dict[str, Any]]] = None,
        collection: Optional[str] = None,
        classify_before_embed: bool = True,
        policies: Optional[List[str]] = None,
    ) -> EmbedResult:
        """Embed documents with governance.

        Classifies documents before embedding and applies policies to
        block or redact sensitive content.

        Args:
            documents: List of document texts to embed
            vector_db: Target vector database (qdrant, pinecone, weaviate, etc.)
            metadata: List of metadata dicts for each document
            collection: Collection/index name in the vector DB
            classify_before_embed: Whether to classify documents first
            policies: Policies to apply

        Returns:
            EmbedResult with embedding statistics

        Example:
            >>> result = gate.embed(
            ...     documents=["John Smith earns $150,000 annually..."],
            ...     metadata=[{"source": "hr_docs", "department": "engineering"}],
            ...     vector_db="qdrant",
            ...     classify_before_embed=True,
            ...     policies=["redact_salary"]
            ... )
            >>> print(f"Embedded {result.embeddings_stored} documents")
        """
        if metadata and len(metadata) != len(documents):
            raise ValueError("metadata list must match documents list length")

        data = {
            "documents": documents,
            "vector_db": vector_db,
            "metadata": metadata or [{} for _ in documents],
            "collection": collection,
            "classify_before_embed": classify_before_embed,
            "policies": policies or [],
        }

        response = self.client.post("/gate/embed", data=data)
        return self._parse_embed_result(response)

    async def aembed(
        self,
        documents: List[str],
        vector_db: str,
        metadata: Optional[List[Dict[str, Any]]] = None,
        collection: Optional[str] = None,
        classify_before_embed: bool = True,
        policies: Optional[List[str]] = None,
    ) -> EmbedResult:
        """Async version of embed."""
        if metadata and len(metadata) != len(documents):
            raise ValueError("metadata list must match documents list length")

        data = {
            "documents": documents,
            "vector_db": vector_db,
            "metadata": metadata or [{} for _ in documents],
            "collection": collection,
            "classify_before_embed": classify_before_embed,
            "policies": policies or [],
        }

        response = await self.client.apost("/gate/embed", data=data)
        return self._parse_embed_result(response)

    def classify(
        self,
        text: str,
        entity_types: Optional[List[str]] = None,
    ) -> List[Classification]:
        """Classify text for sensitive data without applying policies.

        Args:
            text: Text to classify
            entity_types: Optional list of entity types to detect

        Returns:
            List of detected classifications
        """
        data = {
            "text": text,
            "entity_types": entity_types,
        }

        response = self.client.post("/classify", data=data)
        return [Classification(**c) for c in response.get("classifications", [])]

    async def aclassify(
        self,
        text: str,
        entity_types: Optional[List[str]] = None,
    ) -> List[Classification]:
        """Async version of classify."""
        data = {
            "text": text,
            "entity_types": entity_types,
        }

        response = await self.client.apost("/classify", data=data)
        return [Classification(**c) for c in response.get("classifications", [])]

    def _parse_intercept_result(self, response: Dict[str, Any]) -> InterceptResult:
        """Parse intercept API response into InterceptResult."""
        classifications = [
            Classification(**c) for c in response.get("classifications", [])
        ]
        violations = [
            PolicyViolation(**v) for v in response.get("policy_violations", [])
        ]

        return InterceptResult(
            audit_id=response["audit_id"],
            original_query=response["original_query"],
            safe_query=response["safe_query"],
            classifications=classifications,
            policy_violations=violations,
            blocked=response.get("blocked", False),
            block_reason=response.get("block_reason"),
            processing_time_ms=response["processing_time_ms"],
        )

    def _parse_query_result(self, response: Dict[str, Any]) -> QueryResult:
        """Parse query API response into QueryResult."""
        context_chunks = []
        for chunk in response.get("context_chunks", []):
            chunk_classifications = [
                Classification(**c) for c in chunk.get("classifications", [])
            ]
            context_chunks.append(
                ContextChunk(
                    content=chunk["content"],
                    score=chunk["score"],
                    metadata=chunk.get("metadata", {}),
                    source=chunk.get("source"),
                    classifications=chunk_classifications,
                )
            )

        query_classifications = [
            Classification(**c) for c in response.get("query_classifications", [])
        ]
        response_classifications = [
            Classification(**c) for c in response.get("response_classifications", [])
        ]
        violations = [
            PolicyViolation(**v) for v in response.get("policy_violations", [])
        ]

        return QueryResult(
            audit_id=response["audit_id"],
            query=response["query"],
            response=response["response"],
            context_chunks=context_chunks,
            query_classifications=query_classifications,
            response_classifications=response_classifications,
            policy_violations=violations,
            model=response["model"],
            provider=response["provider"],
            tokens_used=response.get("tokens_used"),
            processing_time_ms=response["processing_time_ms"],
        )

    def _parse_embed_result(self, response: Dict[str, Any]) -> EmbedResult:
        """Parse embed API response into EmbedResult."""
        return EmbedResult(
            audit_id=response["audit_id"],
            documents_processed=response["documents_processed"],
            documents_blocked=response.get("documents_blocked", 0),
            classifications_found=response.get("classifications_found", 0),
            embeddings_stored=response["embeddings_stored"],
            vector_db=response["vector_db"],
            processing_time_ms=response["processing_time_ms"],
            blocked_documents=response.get("blocked_documents", []),
        )
