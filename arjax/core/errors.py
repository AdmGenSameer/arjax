"""Structured error handling and alternative provider suggestion engine for Arjax."""

from typing import Any, Dict, List, Optional
from arjax.config.logging import get_logger

logger = get_logger(__name__)


class ArjaxError(Exception):
    """Base exception for all Arjax operations."""
    pass


class ProviderError(ArjaxError):
    """Raised when a package provider operation fails."""
    def __init__(self, message: str, provider: str = "", package: str = ""):
        super().__init__(message)
        self.provider = provider
        self.package = package


class PackageNotFoundError(ProviderError):
    """Raised when a package cannot be located in the specified provider."""
    pass


class InstallFailedError(ProviderError):
    """Raised when package installation execution fails."""
    pass


def suggest_alternative_providers(
    package_name: str,
    failed_provider: str,
    engine: Any
) -> List[Dict[str, str]]:
    """
    Search for the requested package across alternative providers when the primary provider fails.
    
    Returns:
        List of dicts containing alternative package name, provider, and recommended CLI command.
    """
    suggestions: List[Dict[str, str]] = []
    if not engine:
        return suggestions

    try:
        search_res = engine.search(package_name)
        packages = search_res.get("packages", []) if isinstance(search_res, dict) else []

        seen_providers = set()
        for pkg in packages:
            prov = pkg.get("provider", "repo")
            name = pkg.get("name", package_name)

            if prov != failed_provider and prov not in seen_providers:
                seen_providers.add(prov)
                suggestions.append({
                    "name": name,
                    "provider": prov,
                    "version": pkg.get("version", ""),
                    "command": f"arjax install {name} --provider {prov}"
                })
    except Exception as e:
        logger.error(f"Error querying alternative providers for {package_name}: {e}")

    return suggestions
