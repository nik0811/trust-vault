"""Authentication helpers for SecureLens SDK."""

from typing import Dict, Optional
from abc import ABC, abstractmethod


class AuthStrategy(ABC):
    """Base class for authentication strategies."""

    @abstractmethod
    def get_headers(self) -> Dict[str, str]:
        """Return headers to add to requests."""
        pass

    @abstractmethod
    def is_valid(self) -> bool:
        """Check if credentials are valid format."""
        pass


class APIKeyAuth(AuthStrategy):
    """API Key authentication.

    Supports both Bearer token and X-API-Key header formats.
    """

    def __init__(
        self,
        api_key: str,
        header_format: str = "bearer",
    ):
        """Initialize API key authentication.

        Args:
            api_key: The API key (should start with 'sl_' for SecureLens keys)
            header_format: Either 'bearer' for Authorization header or 'x-api-key'
        """
        self.api_key = api_key
        self.header_format = header_format.lower()

        if self.header_format not in ("bearer", "x-api-key"):
            raise ValueError("header_format must be 'bearer' or 'x-api-key'")

    def get_headers(self) -> Dict[str, str]:
        """Return authentication headers."""
        if self.header_format == "bearer":
            return {"Authorization": f"Bearer {self.api_key}"}
        return {"X-API-Key": self.api_key}

    def is_valid(self) -> bool:
        """Check if API key has valid format."""
        if not self.api_key:
            return False
        return self.api_key.startswith("sl_") and len(self.api_key) >= 20


class TenantAuth(AuthStrategy):
    """Multi-tenant authentication with tenant ID."""

    def __init__(
        self,
        api_key: str,
        tenant_id: str,
        header_format: str = "bearer",
    ):
        """Initialize tenant authentication.

        Args:
            api_key: The API key
            tenant_id: The tenant identifier
            header_format: Either 'bearer' or 'x-api-key'
        """
        self._api_key_auth = APIKeyAuth(api_key, header_format)
        self.tenant_id = tenant_id

    def get_headers(self) -> Dict[str, str]:
        """Return authentication headers including tenant ID."""
        headers = self._api_key_auth.get_headers()
        headers["X-Tenant-ID"] = self.tenant_id
        return headers

    def is_valid(self) -> bool:
        """Check if credentials are valid."""
        return self._api_key_auth.is_valid() and bool(self.tenant_id)


def create_auth(
    api_key: str,
    tenant_id: Optional[str] = None,
    header_format: str = "bearer",
) -> AuthStrategy:
    """Factory function to create appropriate auth strategy.

    Args:
        api_key: The API key
        tenant_id: Optional tenant ID for multi-tenant setups
        header_format: Header format to use

    Returns:
        Appropriate AuthStrategy instance
    """
    if tenant_id:
        return TenantAuth(api_key, tenant_id, header_format)
    return APIKeyAuth(api_key, header_format)
