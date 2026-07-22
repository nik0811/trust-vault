"""Pydantic models for SecureLens SDK requests and responses."""

from typing import Optional, List, Dict, Any
from datetime import datetime
from enum import Enum
from pydantic import BaseModel, Field


class SensitivityLevel(str, Enum):
    """Data sensitivity levels."""

    PUBLIC = "public"
    INTERNAL = "internal"
    CONFIDENTIAL = "confidential"
    RESTRICTED = "restricted"


class ClassificationType(str, Enum):
    """Types of data classifications."""

    PII = "pii"
    PHI = "phi"
    PCI = "pci"
    FINANCIAL = "financial"
    LEGAL = "legal"
    CUSTOM = "custom"


class PolicyAction(str, Enum):
    """Actions that can be taken by policies."""

    ALLOW = "allow"
    BLOCK = "block"
    REDACT = "redact"
    MASK = "mask"
    WARN = "warn"


class Classification(BaseModel):
    """A detected classification in the content."""

    entity_type: str = Field(..., description="Type of entity detected (e.g., 'PERSON', 'SSN')")
    value: str = Field(..., description="The detected value")
    start: int = Field(..., description="Start position in text")
    end: int = Field(..., description="End position in text")
    confidence: float = Field(..., ge=0.0, le=1.0, description="Confidence score")
    classification_type: ClassificationType = Field(
        default=ClassificationType.PII, description="Category of classification"
    )
    sensitivity: SensitivityLevel = Field(
        default=SensitivityLevel.CONFIDENTIAL, description="Sensitivity level"
    )

    class Config:
        use_enum_values = True


class PolicyViolation(BaseModel):
    """A policy violation detected in the content."""

    policy_id: str = Field(..., description="ID of the violated policy")
    policy_name: str = Field(..., description="Name of the violated policy")
    action: PolicyAction = Field(..., description="Action taken")
    reason: str = Field(..., description="Reason for the violation")
    classifications: List[Classification] = Field(
        default_factory=list, description="Classifications that triggered the violation"
    )

    class Config:
        use_enum_values = True


class InterceptResult(BaseModel):
    """Result of intercepting and classifying a query."""

    audit_id: str = Field(..., description="Unique ID for audit tracking")
    original_query: str = Field(..., description="The original query")
    safe_query: str = Field(..., description="The query after redaction/masking")
    classifications: List[Classification] = Field(
        default_factory=list, description="Detected classifications"
    )
    policy_violations: List[PolicyViolation] = Field(
        default_factory=list, description="Policy violations"
    )
    blocked: bool = Field(default=False, description="Whether the query was blocked")
    block_reason: Optional[str] = Field(None, description="Reason for blocking")
    processing_time_ms: int = Field(..., description="Processing time in milliseconds")
    timestamp: datetime = Field(default_factory=datetime.utcnow)

    class Config:
        use_enum_values = True


class ContextChunk(BaseModel):
    """A chunk of context retrieved from vector DB."""

    content: str = Field(..., description="The text content")
    score: float = Field(..., description="Relevance score")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="Chunk metadata")
    source: Optional[str] = Field(None, description="Source document")
    classifications: List[Classification] = Field(
        default_factory=list, description="Classifications in this chunk"
    )


class QueryResult(BaseModel):
    """Result of a full query through the AI Gate."""

    audit_id: str = Field(..., description="Unique ID for audit tracking")
    query: str = Field(..., description="The processed query")
    response: str = Field(..., description="The LLM response")
    context_chunks: List[ContextChunk] = Field(
        default_factory=list, description="Retrieved context"
    )
    query_classifications: List[Classification] = Field(
        default_factory=list, description="Classifications in query"
    )
    response_classifications: List[Classification] = Field(
        default_factory=list, description="Classifications in response"
    )
    policy_violations: List[PolicyViolation] = Field(
        default_factory=list, description="Policy violations"
    )
    model: str = Field(..., description="LLM model used")
    provider: str = Field(..., description="LLM provider")
    tokens_used: Optional[int] = Field(None, description="Total tokens used")
    processing_time_ms: int = Field(..., description="Total processing time")
    timestamp: datetime = Field(default_factory=datetime.utcnow)

    class Config:
        use_enum_values = True


class EmbedResult(BaseModel):
    """Result of embedding documents with governance."""

    audit_id: str = Field(..., description="Unique ID for audit tracking")
    documents_processed: int = Field(..., description="Number of documents processed")
    documents_blocked: int = Field(default=0, description="Number blocked by policy")
    classifications_found: int = Field(default=0, description="Total classifications found")
    embeddings_stored: int = Field(..., description="Number of embeddings stored")
    vector_db: str = Field(..., description="Target vector database")
    processing_time_ms: int = Field(..., description="Processing time")
    blocked_documents: List[Dict[str, Any]] = Field(
        default_factory=list, description="Details of blocked documents"
    )
    timestamp: datetime = Field(default_factory=datetime.utcnow)


class InterceptRequest(BaseModel):
    """Request to intercept and classify a query."""

    query: str = Field(..., min_length=1, description="The query to intercept")
    context_source: Optional[str] = Field(None, description="Vector DB source")
    policies: List[str] = Field(default_factory=list, description="Policies to apply")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="Additional metadata")


class QueryRequest(BaseModel):
    """Request for a full query through AI Gate."""

    query: str = Field(..., min_length=1, description="The query")
    llm_provider: str = Field(..., description="LLM provider (openai, anthropic, etc.)")
    model: str = Field(..., description="Model name")
    context_source: Optional[str] = Field(None, description="Vector DB source")
    top_k: int = Field(default=5, ge=1, le=100, description="Number of context chunks")
    policies: List[str] = Field(default_factory=list, description="Policies to apply")
    system_prompt: Optional[str] = Field(None, description="Custom system prompt")
    temperature: float = Field(default=0.7, ge=0.0, le=2.0)
    max_tokens: Optional[int] = Field(None, ge=1)
    metadata: Dict[str, Any] = Field(default_factory=dict)


class EmbedRequest(BaseModel):
    """Request to embed documents with governance."""

    documents: List[str] = Field(..., min_length=1, description="Documents to embed")
    metadata: List[Dict[str, Any]] = Field(default_factory=list, description="Document metadata")
    vector_db: str = Field(..., description="Target vector database")
    collection: Optional[str] = Field(None, description="Collection/index name")
    classify_before_embed: bool = Field(default=True, description="Classify before embedding")
    policies: List[str] = Field(default_factory=list, description="Policies to apply")
