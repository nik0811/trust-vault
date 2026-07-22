"""Custom exceptions for SecureLens SDK."""

from typing import Optional, List, Any


class SecureLensError(Exception):
    """Base exception for all SecureLens errors."""

    def __init__(
        self,
        message: str,
        status_code: Optional[int] = None,
        response_body: Optional[Any] = None,
    ):
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.response_body = response_body

    def __str__(self) -> str:
        if self.status_code:
            return f"[{self.status_code}] {self.message}"
        return self.message


class SecureLensAuthError(SecureLensError):
    """Raised when authentication fails (invalid or expired API key)."""

    pass


class SecureLensRateLimitError(SecureLensError):
    """Raised when rate limit is exceeded."""

    def __init__(
        self,
        message: str,
        retry_after: Optional[int] = None,
        status_code: int = 429,
        response_body: Optional[Any] = None,
    ):
        super().__init__(message, status_code, response_body)
        self.retry_after = retry_after


class SecureLensPolicyError(SecureLensError):
    """Raised when a query is blocked by policy."""

    def __init__(
        self,
        message: str,
        policy_name: Optional[str] = None,
        violations: Optional[List[str]] = None,
        status_code: int = 403,
        response_body: Optional[Any] = None,
    ):
        super().__init__(message, status_code, response_body)
        self.policy_name = policy_name
        self.violations = violations or []


class SecureLensConnectionError(SecureLensError):
    """Raised when network connection fails."""

    pass


class SecureLensValidationError(SecureLensError):
    """Raised when request validation fails."""

    pass
