"""LangChain integration for SecureLens AI Gate."""

from typing import Any, Dict, List, Optional, Union
from securelens.gate import AIGate
from securelens.models import InterceptResult

try:
    from langchain.callbacks.base import BaseCallbackHandler
    from langchain.schema import LLMResult, Document
    LANGCHAIN_AVAILABLE = True
except ImportError:
    LANGCHAIN_AVAILABLE = False
    BaseCallbackHandler = object


class SecureLensLangChainCallback(BaseCallbackHandler if LANGCHAIN_AVAILABLE else object):
    """LangChain callback handler for SecureLens AI Gate.

    Intercepts queries and responses in LangChain chains to apply
    classification and governance policies.

    Example:
        >>> from securelens import SecureLensClient, AIGate
        >>> from securelens.integrations import SecureLensLangChainCallback
        >>> from langchain.chains import RetrievalQA
        >>>
        >>> client = SecureLensClient(api_key="sl_...")
        >>> gate = AIGate(client)
        >>> callback = SecureLensLangChainCallback(gate)
        >>>
        >>> chain = RetrievalQA.from_chain_type(
        ...     llm=llm,
        ...     retriever=retriever,
        ...     callbacks=[callback]
        ... )
    """

    def __init__(
        self,
        gate: AIGate,
        policies: Optional[List[str]] = None,
        block_on_violation: bool = False,
        redact_queries: bool = True,
        redact_responses: bool = True,
        metadata: Optional[Dict[str, Any]] = None,
    ):
        """Initialize the callback handler.

        Args:
            gate: AIGate instance for classification
            policies: Default policies to apply
            block_on_violation: Whether to raise exception on policy violation
            redact_queries: Whether to redact sensitive data in queries
            redact_responses: Whether to redact sensitive data in responses
            metadata: Additional metadata for audit logging
        """
        if not LANGCHAIN_AVAILABLE:
            raise ImportError(
                "LangChain is required for this integration. "
                "Install it with: pip install langchain"
            )

        self.gate = gate
        self.policies = policies or []
        self.block_on_violation = block_on_violation
        self.redact_queries = redact_queries
        self.redact_responses = redact_responses
        self.metadata = metadata or {}

        self._last_intercept_result: Optional[InterceptResult] = None
        self._audit_ids: List[str] = []

    @property
    def last_intercept_result(self) -> Optional[InterceptResult]:
        """Get the last intercept result."""
        return self._last_intercept_result

    @property
    def audit_ids(self) -> List[str]:
        """Get all audit IDs from this session."""
        return self._audit_ids.copy()

    def on_llm_start(
        self,
        serialized: Dict[str, Any],
        prompts: List[str],
        **kwargs: Any,
    ) -> None:
        """Called when LLM starts processing.

        Intercepts the prompts and applies classification/policies.
        """
        for i, prompt in enumerate(prompts):
            result = self.gate.intercept(
                query=prompt,
                policies=self.policies,
                metadata={**self.metadata, "source": "langchain", "prompt_index": i},
            )

            self._last_intercept_result = result
            self._audit_ids.append(result.audit_id)

            if result.blocked and self.block_on_violation:
                from securelens.exceptions import SecureLensPolicyError
                raise SecureLensPolicyError(
                    f"Query blocked by policy: {result.block_reason}",
                    violations=[v.reason for v in result.policy_violations],
                )

            if self.redact_queries and result.safe_query != result.original_query:
                prompts[i] = result.safe_query

    def on_llm_end(self, response: "LLMResult", **kwargs: Any) -> None:
        """Called when LLM finishes processing.

        Classifies the response for sensitive data.
        """
        if not self.redact_responses:
            return

        for generation_list in response.generations:
            for generation in generation_list:
                if hasattr(generation, "text") and generation.text:
                    result = self.gate.intercept(
                        query=generation.text,
                        policies=self.policies,
                        metadata={**self.metadata, "source": "langchain_response"},
                    )
                    self._audit_ids.append(result.audit_id)

    def on_chain_start(
        self,
        serialized: Dict[str, Any],
        inputs: Dict[str, Any],
        **kwargs: Any,
    ) -> None:
        """Called when chain starts."""
        pass

    def on_chain_end(self, outputs: Dict[str, Any], **kwargs: Any) -> None:
        """Called when chain ends."""
        pass

    def on_retriever_start(
        self,
        serialized: Dict[str, Any],
        query: str,
        **kwargs: Any,
    ) -> None:
        """Called when retriever starts.

        Intercepts the retrieval query.
        """
        result = self.gate.intercept(
            query=query,
            policies=self.policies,
            metadata={**self.metadata, "source": "langchain_retriever"},
        )
        self._last_intercept_result = result
        self._audit_ids.append(result.audit_id)

    def on_retriever_end(
        self,
        documents: List["Document"],
        **kwargs: Any,
    ) -> None:
        """Called when retriever ends.

        Classifies retrieved documents.
        """
        for doc in documents:
            if hasattr(doc, "page_content") and doc.page_content:
                result = self.gate.intercept(
                    query=doc.page_content,
                    policies=self.policies,
                    metadata={**self.metadata, "source": "langchain_retrieved_doc"},
                )
                self._audit_ids.append(result.audit_id)


class SecureLensRetriever:
    """A wrapper retriever that applies SecureLens governance.

    Wraps an existing LangChain retriever to add classification
    and policy enforcement.

    Example:
        >>> from securelens.integrations import SecureLensRetriever
        >>>
        >>> secure_retriever = SecureLensRetriever(
        ...     retriever=base_retriever,
        ...     gate=gate,
        ...     policies=["redact_pii"]
        ... )
    """

    def __init__(
        self,
        retriever: Any,
        gate: AIGate,
        policies: Optional[List[str]] = None,
        redact_content: bool = True,
    ):
        """Initialize the secure retriever.

        Args:
            retriever: Base LangChain retriever
            gate: AIGate instance
            policies: Policies to apply
            redact_content: Whether to redact document content
        """
        if not LANGCHAIN_AVAILABLE:
            raise ImportError(
                "LangChain is required for this integration. "
                "Install it with: pip install langchain"
            )

        self.retriever = retriever
        self.gate = gate
        self.policies = policies or []
        self.redact_content = redact_content

    def get_relevant_documents(self, query: str) -> List["Document"]:
        """Retrieve documents with governance applied."""
        query_result = self.gate.intercept(
            query=query,
            policies=self.policies,
        )

        safe_query = query_result.safe_query if self.redact_content else query
        documents = self.retriever.get_relevant_documents(safe_query)

        if self.redact_content:
            for doc in documents:
                if hasattr(doc, "page_content") and doc.page_content:
                    doc_result = self.gate.intercept(
                        query=doc.page_content,
                        policies=self.policies,
                    )
                    doc.page_content = doc_result.safe_query

        return documents

    async def aget_relevant_documents(self, query: str) -> List["Document"]:
        """Async retrieve documents with governance applied."""
        query_result = await self.gate.aintercept(
            query=query,
            policies=self.policies,
        )

        safe_query = query_result.safe_query if self.redact_content else query

        if hasattr(self.retriever, "aget_relevant_documents"):
            documents = await self.retriever.aget_relevant_documents(safe_query)
        else:
            documents = self.retriever.get_relevant_documents(safe_query)

        if self.redact_content:
            for doc in documents:
                if hasattr(doc, "page_content") and doc.page_content:
                    doc_result = await self.gate.aintercept(
                        query=doc.page_content,
                        policies=self.policies,
                    )
                    doc.page_content = doc_result.safe_query

        return documents
