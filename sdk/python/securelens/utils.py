"""Utility functions for SecureLens SDK."""

import re
from typing import Optional, Dict, Any, List
from datetime import datetime


def mask_value(value: str, mask_char: str = "*", visible_chars: int = 4) -> str:
    """Mask a sensitive value, keeping some characters visible.

    Args:
        value: The value to mask
        mask_char: Character to use for masking
        visible_chars: Number of characters to keep visible at the end

    Returns:
        Masked string
    """
    if len(value) <= visible_chars:
        return mask_char * len(value)
    return mask_char * (len(value) - visible_chars) + value[-visible_chars:]


def redact_text(
    text: str,
    classifications: List[Dict[str, Any]],
    replacement: str = "[REDACTED]",
) -> str:
    """Redact classified entities from text.

    Args:
        text: Original text
        classifications: List of classification dicts with 'start' and 'end' keys
        replacement: Replacement string for redacted content

    Returns:
        Text with redacted content
    """
    sorted_classifications = sorted(classifications, key=lambda x: x["start"], reverse=True)

    result = text
    for classification in sorted_classifications:
        start = classification["start"]
        end = classification["end"]
        entity_type = classification.get("entity_type", "UNKNOWN")
        result = result[:start] + f"[{entity_type}]" + result[end:]

    return result


def validate_api_key(api_key: str) -> bool:
    """Validate API key format.

    Args:
        api_key: The API key to validate

    Returns:
        True if valid format
    """
    if not api_key:
        return False
    pattern = r"^sl_[a-zA-Z0-9]{16,}$"
    return bool(re.match(pattern, api_key))


def parse_retry_after(headers: Dict[str, str]) -> Optional[int]:
    """Parse Retry-After header value.

    Args:
        headers: Response headers

    Returns:
        Seconds to wait, or None if not present
    """
    retry_after = headers.get("Retry-After") or headers.get("retry-after")
    if not retry_after:
        return None

    try:
        return int(retry_after)
    except ValueError:
        try:
            retry_date = datetime.strptime(retry_after, "%a, %d %b %Y %H:%M:%S GMT")
            delta = retry_date - datetime.utcnow()
            return max(0, int(delta.total_seconds()))
        except ValueError:
            return None


def build_query_params(params: Dict[str, Any]) -> Dict[str, str]:
    """Build query parameters, filtering None values.

    Args:
        params: Dictionary of parameters

    Returns:
        Filtered dictionary with string values
    """
    result = {}
    for key, value in params.items():
        if value is None:
            continue
        if isinstance(value, bool):
            result[key] = str(value).lower()
        elif isinstance(value, (list, tuple)):
            result[key] = ",".join(str(v) for v in value)
        else:
            result[key] = str(value)
    return result


def chunk_list(items: List[Any], chunk_size: int) -> List[List[Any]]:
    """Split a list into chunks.

    Args:
        items: List to chunk
        chunk_size: Maximum size of each chunk

    Returns:
        List of chunks
    """
    return [items[i : i + chunk_size] for i in range(0, len(items), chunk_size)]


def format_duration_ms(ms: int) -> str:
    """Format milliseconds as human-readable duration.

    Args:
        ms: Duration in milliseconds

    Returns:
        Formatted string
    """
    if ms < 1000:
        return f"{ms}ms"
    if ms < 60000:
        return f"{ms / 1000:.2f}s"
    minutes = ms // 60000
    seconds = (ms % 60000) / 1000
    return f"{minutes}m {seconds:.1f}s"
