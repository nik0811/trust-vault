"""
SecureLens AI Gate Python SDK

A Python SDK for integrating with SecureLens AI Gate to add governance,
classification, and policy enforcement to RAG applications.
"""

from securelens.client import SecureLensClient
from securelens.gate import AIGate
from securelens.auth import APIKeyAuth
from securelens.exceptions import (
    SecureLensError,
    SecureLensAuthError,
    SecureLensRateLimitError,
    SecureLensPolicyError,
    SecureLensConnectionError,
    SecureLensValidationError,
)
from securelens.models import (
    InterceptResult,
    QueryResult,
    Classification,
    PolicyViolation,
    EmbedResult,
)

__version__ = "0.1.0"
__all__ = [
    "SecureLensClient",
    "AIGate",
    "APIKeyAuth",
    "SecureLensError",
    "SecureLensAuthError",
    "SecureLensRateLimitError",
    "SecureLensPolicyError",
    "SecureLensConnectionError",
    "SecureLensValidationError",
    "InterceptResult",
    "QueryResult",
    "Classification",
    "PolicyViolation",
    "EmbedResult",
]
