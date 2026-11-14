import pytest
from unittest.mock import MagicMock
from arjax.core.errors import (
    ArjaxError,
    ProviderError,
    PackageNotFoundError,
    InstallFailedError,
    suggest_alternative_providers,
)


def test_custom_exception_hierarchy():
    err = InstallFailedError("Installation failed", provider="repo", package="testpkg")
    assert isinstance(err, ProviderError)
    assert isinstance(err, ArjaxError)
    assert err.provider == "repo"
    assert err.package == "testpkg"


def test_suggest_alternative_providers():
    mock_engine = MagicMock()
    mock_engine.search.return_value = {
        "packages": [
            {"name": "testpkg", "provider": "repo", "version": "1.0"},
            {"name": "testpkg", "provider": "flatpak", "version": "1.0-flathub"},
            {"name": "testpkg", "provider": "aur", "version": "1.0-1"},
        ]
    }

    suggestions = suggest_alternative_providers("testpkg", failed_provider="repo", engine=mock_engine)
    providers = [s["provider"] for s in suggestions]
    assert "flatpak" in providers
    assert "aur" in providers
    assert "repo" not in providers
