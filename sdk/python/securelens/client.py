"""Main SecureLens client for API communication."""

import httpx
from typing import Optional, Dict, Any, Union
from urllib.parse import urljoin

from securelens.auth import AuthStrategy, APIKeyAuth, create_auth
from securelens.exceptions import (
    SecureLensError,
    SecureLensAuthError,
    SecureLensRateLimitError,
    SecureLensPolicyError,
    SecureLensConnectionError,
    SecureLensValidationError,
)
from securelens.utils import parse_retry_after


class SecureLensClient:
    """Client for communicating with SecureLens API.

    Example:
        >>> client = SecureLensClient(
        ...     api_key="sl_your_api_key",
        ...     base_url="https://api.securelens.ai"
        ... )
        >>> # Use with AIGate
        >>> from securelens import AIGate
        >>> gate = AIGate(client)
    """

    DEFAULT_BASE_URL = "https://api.securelens.ai"
    DEFAULT_TIMEOUT = 30.0
    API_VERSION = "v1"

    def __init__(
        self,
        api_key: str,
        base_url: Optional[str] = None,
        tenant_id: Optional[str] = None,
        timeout: float = DEFAULT_TIMEOUT,
        auth_header_format: str = "bearer",
        verify_ssl: bool = True,
        custom_headers: Optional[Dict[str, str]] = None,
    ):
        """Initialize SecureLens client.

        Args:
            api_key: SecureLens API key (starts with 'sl_')
            base_url: API base URL (default: https://api.securelens.ai)
            tenant_id: Optional tenant ID for multi-tenant deployments
            timeout: Request timeout in seconds
            auth_header_format: 'bearer' or 'x-api-key'
            verify_ssl: Whether to verify SSL certificates
            custom_headers: Additional headers to include in requests
        """
        self.base_url = (base_url or self.DEFAULT_BASE_URL).rstrip("/")
        self.timeout = timeout
        self.verify_ssl = verify_ssl

        self._auth = create_auth(api_key, tenant_id, auth_header_format)
        self._custom_headers = custom_headers or {}

        self._client: Optional[httpx.Client] = None
        self._async_client: Optional[httpx.AsyncClient] = None

    @property
    def _headers(self) -> Dict[str, str]:
        """Build request headers."""
        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": "securelens-python/0.1.0",
        }
        headers.update(self._auth.get_headers())
        headers.update(self._custom_headers)
        return headers

    def _get_client(self) -> httpx.Client:
        """Get or create sync HTTP client."""
        if self._client is None:
            self._client = httpx.Client(
                base_url=self.base_url,
                headers=self._headers,
                timeout=self.timeout,
                verify=self.verify_ssl,
            )
        return self._client

    def _get_async_client(self) -> httpx.AsyncClient:
        """Get or create async HTTP client."""
        if self._async_client is None:
            self._async_client = httpx.AsyncClient(
                base_url=self.base_url,
                headers=self._headers,
                timeout=self.timeout,
                verify=self.verify_ssl,
            )
        return self._async_client

    def _build_url(self, endpoint: str) -> str:
        """Build full URL for endpoint."""
        if not endpoint.startswith("/"):
            endpoint = f"/{endpoint}"
        return f"/api/{self.API_VERSION}{endpoint}"

    def _handle_response(self, response: httpx.Response) -> Dict[str, Any]:
        """Handle API response and raise appropriate exceptions."""
        if response.status_code == 200:
            return response.json()

        if response.status_code == 201:
            return response.json()

        try:
            error_body = response.json()
            error_message = error_body.get("error", {}).get("message", response.text)
        except Exception:
            error_body = None
            error_message = response.text

        if response.status_code == 401:
            raise SecureLensAuthError(
                "Authentication failed. Check your API key.",
                status_code=401,
                response_body=error_body,
            )

        if response.status_code == 403:
            policy_name = error_body.get("policy_name") if error_body else None
            violations = error_body.get("violations") if error_body else None
            raise SecureLensPolicyError(
                error_message or "Request blocked by policy",
                policy_name=policy_name,
                violations=violations,
                status_code=403,
                response_body=error_body,
            )

        if response.status_code == 422:
            raise SecureLensValidationError(
                error_message or "Validation error",
                status_code=422,
                response_body=error_body,
            )

        if response.status_code == 429:
            retry_after = parse_retry_after(dict(response.headers))
            raise SecureLensRateLimitError(
                "Rate limit exceeded",
                retry_after=retry_after,
                status_code=429,
                response_body=error_body,
            )

        raise SecureLensError(
            error_message or f"API error: {response.status_code}",
            status_code=response.status_code,
            response_body=error_body,
        )

    def request(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make a synchronous API request.

        Args:
            method: HTTP method
            endpoint: API endpoint
            data: Request body
            params: Query parameters

        Returns:
            Response data

        Raises:
            SecureLensError: On API errors
            SecureLensConnectionError: On network errors
        """
        url = self._build_url(endpoint)
        client = self._get_client()

        try:
            response = client.request(
                method=method,
                url=url,
                json=data,
                params=params,
            )
            return self._handle_response(response)
        except httpx.ConnectError as e:
            raise SecureLensConnectionError(f"Connection failed: {e}")
        except httpx.TimeoutException as e:
            raise SecureLensConnectionError(f"Request timed out: {e}")

    async def arequest(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make an asynchronous API request.

        Args:
            method: HTTP method
            endpoint: API endpoint
            data: Request body
            params: Query parameters

        Returns:
            Response data

        Raises:
            SecureLensError: On API errors
            SecureLensConnectionError: On network errors
        """
        url = self._build_url(endpoint)
        client = self._get_async_client()

        try:
            response = await client.request(
                method=method,
                url=url,
                json=data,
                params=params,
            )
            return self._handle_response(response)
        except httpx.ConnectError as e:
            raise SecureLensConnectionError(f"Connection failed: {e}")
        except httpx.TimeoutException as e:
            raise SecureLensConnectionError(f"Request timed out: {e}")

    def get(
        self, endpoint: str, params: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Make a GET request."""
        return self.request("GET", endpoint, params=params)

    def post(
        self,
        endpoint: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make a POST request."""
        return self.request("POST", endpoint, data=data, params=params)

    async def aget(
        self, endpoint: str, params: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Make an async GET request."""
        return await self.arequest("GET", endpoint, params=params)

    async def apost(
        self,
        endpoint: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make an async POST request."""
        return await self.arequest("POST", endpoint, data=data, params=params)

    def health_check(self) -> bool:
        """Check if the API is healthy.

        Returns:
            True if healthy
        """
        try:
            response = self.get("/health")
            return response.get("status") == "healthy"
        except SecureLensError:
            return False

    async def ahealth_check(self) -> bool:
        """Async health check."""
        try:
            response = await self.aget("/health")
            return response.get("status") == "healthy"
        except SecureLensError:
            return False

    def close(self) -> None:
        """Close the HTTP client."""
        if self._client:
            self._client.close()
            self._client = None

    async def aclose(self) -> None:
        """Close the async HTTP client."""
        if self._async_client:
            await self._async_client.aclose()
            self._async_client = None

    def __enter__(self) -> "SecureLensClient":
        return self

    def __exit__(self, *args) -> None:
        self.close()

    async def __aenter__(self) -> "SecureLensClient":
        return self

    async def __aexit__(self, *args) -> None:
        await self.aclose()
