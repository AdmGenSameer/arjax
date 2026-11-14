"""
Unit and integration tests for Pop!_OS support and automated GitHub / Flatpak / AppImage resolution.
"""

import pytest
from unittest.mock import Mock, patch, MagicMock
from pathlib import Path
import tempfile

from arjax.config.base import DISTRO_MAP, SUPPORTED_PLATFORMS
from arjax.installation.models import ProviderConfig, ProviderResult, Recipe
from arjax.installation.providers import (
    RepositoryProvider,
    FlatpakProvider,
    GithubReleaseProvider,
    AppImageProvider,
    ProviderManager,
    detect_distro_family,
    default_package_manager,
    is_package_in_repo,
    create_desktop_entry,
)
from arjax.integrations.github import (
    search_github_repositories,
    get_latest_github_release,
    select_best_github_asset,
    validate_github_url,
)
from arjax.installation.orchestrator import InstallationOrchestrator


class TestPopOSDistroDetection:
    """Verify Pop!_OS distribution recognition and package manager resolution."""

    def test_pop_os_distro_mapping(self):
        """Pop!_OS variations must map to the debian family and be in supported platforms."""
        assert "pop" in SUPPORTED_PLATFORMS
        assert "pop-os" in SUPPORTED_PLATFORMS
        assert DISTRO_MAP["pop"] == "debian"
        assert DISTRO_MAP["pop-os"] == "debian"
        assert DISTRO_MAP["pop_os"] == "debian"

    @patch("distro.id", return_value="pop")
    def test_pop_os_default_package_manager(self, mock_distro):
        """Pop!_OS must use apt as default package manager."""
        assert detect_distro_family() == "debian"
        assert default_package_manager() == "apt"


class TestGitHubAssetSelection:
    """Verify intelligent asset selection for Pop!_OS and cross-distro platforms."""

    @pytest.fixture
    def zapfast_assets(self):
        return [
            {"name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt"},
            {"name": "zapfast-0.21.1-1.x86_64.rpm", "browser_download_url": "https://example.com/zapfast.rpm"},
            {"name": "zapfast-0.21.1-x86_64.AppImage", "browser_download_url": "https://example.com/zapfast-x86_64.AppImage"},
            {"name": "zapfast-0.21.1-aarch64.AppImage", "browser_download_url": "https://example.com/zapfast-aarch64.AppImage"},
            {"name": "zapfast-v0.21.1-x86_64-pc-windows-msvc.msi", "browser_download_url": "https://example.com/zapfast.msi"},
            {"name": "zapfast_0.21.1_amd64.deb", "browser_download_url": "https://example.com/zapfast_amd64.deb"},
            {"name": "zapfast_0.21.1_arm64.deb", "browser_download_url": "https://example.com/zapfast_arm64.deb"},
            {"name": "zapfast-v0.21.1-x86_64-unknown-linux-gnu.tar.gz", "browser_download_url": "https://example.com/zapfast.tar.gz"},
        ]

    def test_pop_os_prefers_deb(self, zapfast_assets):
        """On Pop!_OS (debian family, x86_64), .deb package matching amd64 must be chosen."""
        asset = select_best_github_asset(zapfast_assets, distro_family="debian", machine_arch="x86_64")
        assert asset is not None
        assert asset["name"] == "zapfast_0.21.1_amd64.deb"

    def test_pop_os_appimage_preference(self, zapfast_assets):
        """When AppImage is preferred, select x86_64 AppImage asset."""
        asset = select_best_github_asset(zapfast_assets, distro_family="debian", machine_arch="x86_64", preferred_format="appimage")
        assert asset is not None
        assert asset["name"] == "zapfast-0.21.1-x86_64.AppImage"

    def test_arch_prefers_appimage(self, zapfast_assets):
        """On Arch Linux, select AppImage asset."""
        asset = select_best_github_asset(zapfast_assets, distro_family="arch", machine_arch="x86_64")
        assert asset is not None
        assert asset["name"] == "zapfast-0.21.1-x86_64.AppImage"

    def test_arm64_architecture_filtering(self, zapfast_assets):
        """On arm64/aarch64 system, x86_64 assets must be ignored."""
        asset = select_best_github_asset(zapfast_assets, distro_family="debian", machine_arch="aarch64")
        assert asset is not None
        assert asset["name"] == "zapfast_0.21.1_arm64.deb"


class TestDynamicProviders:
    """Verify fallback and automated discovery behavior."""

    @patch("subprocess.run")
    def test_repository_provider_skips_when_not_in_repo(self, mock_run):
        """RepositoryProvider must skip when package is not in native repositories."""
        # Simulate apt-cache show returning exit code 100 (package not found)
        mock_run.return_value = Mock(returncode=100, stdout="", stderr="Unable to locate package")
        
        provider = RepositoryProvider()
        result = provider.install("zapfast", recipe=None)
        
        assert result.skipped
        assert "not found in" in result.message

    @patch("arjax.package_management.command_gen.check_command_availability", return_value=True)
    def test_flatpak_supports_without_recipe_if_available(self, mock_check):
        """FlatpakProvider supports packages when flatpak binary exists."""
        provider = FlatpakProvider()
        assert provider.supports("zapfast", recipe=None)

    @patch("arjax.search.flatpak.search_flatpak", return_value=[])
    def test_flatpak_skips_when_app_not_found(self, mock_search):
        """FlatpakProvider cleanly skips when package is not on Flathub."""
        provider = FlatpakProvider()
        result = provider.install("nonexistent-app", recipe=None)
        assert result.skipped
        assert "No Flatpak app found" in result.message

    @patch("arjax.integrations.github.search_github_repositories")
    @patch("arjax.integrations.github.get_latest_github_release")
    def test_github_provider_automated_resolution(self, mock_release, mock_search):
        """GithubReleaseProvider searches GitHub and selects release asset automatically."""
        mock_search.return_value = [
            {"name": "zapfast", "full_name": "crmne/zapfast", "stars": 4500}
        ]
        mock_release.return_value = {
            "tag_name": "v0.21.1",
            "assets": [
                {"name": "zapfast_0.21.1_amd64.deb", "browser_download_url": "https://example.com/zapfast.deb"}
            ]
        }

        provider = GithubReleaseProvider()
        with patch.object(provider, "_install_downloaded_binary") as mock_dl:
            mock_dl.return_value = ProviderResult(
                provider_name="github",
                package="zapfast",
                success=True,
                message="Installed deb via apt"
            )
            result = provider.install("zapfast", recipe=None)
            assert result.success
            mock_dl.assert_called_once()


class TestDesktopIntegration:
    """Verify XDG Desktop entry generation for AppImage and binaries."""

    def test_create_desktop_entry(self, tmp_path):
        """Desktop file must be created with valid XDG fields."""
        with patch("pathlib.Path.home", return_value=tmp_path):
            exec_file = tmp_path / "bin" / "zapfast"
            exec_file.parent.mkdir(parents=True, exist_ok=True)
            exec_file.touch()

            desktop_file = create_desktop_entry(
                package_name="zapfast",
                exec_path=exec_file,
                display_name="ZapFast WhatsApp",
                comment="Native WhatsApp client"
            )

            assert desktop_file.exists()
            content = desktop_file.read_text()
            assert "Type=Application" in content
            assert "Name=ZapFast WhatsApp" in content
            assert f"Exec={exec_file} %U" in content
            assert "Comment=Native WhatsApp client" in content


class TestPopOSFullFallbackChain:
    """Verify the orchestrator fallback chain: apt -> flatpak -> github."""

    @patch("arjax.installation.providers.detect_distro_family", return_value="debian")
    @patch("arjax.installation.providers.default_package_manager", return_value="apt")
    def test_orchestrator_fallbacks_to_github(self, mock_pm, mock_distro):
        """When apt and flatpak don't have the package, it cleanly falls back to GitHub."""
        repo_prov = Mock(spec=RepositoryProvider)
        repo_prov.name = "repository"
        repo_prov.priority = 10
        repo_prov.supports.return_value = True
        repo_prov.install.return_value = ProviderResult(
            provider_name="repository",
            package="zapfast",
            success=False,
            skipped=True,
            message="Package not found in apt repository"
        )

        flatpak_prov = Mock(spec=FlatpakProvider)
        flatpak_prov.name = "flatpak"
        flatpak_prov.priority = 40
        flatpak_prov.supports.return_value = True
        flatpak_prov.install.return_value = ProviderResult(
            provider_name="flatpak",
            package="zapfast",
            success=False,
            skipped=True,
            message="No Flatpak app found"
        )

        github_prov = Mock(spec=GithubReleaseProvider)
        github_prov.name = "github"
        github_prov.priority = 45
        github_prov.supports.return_value = True
        github_prov.install.return_value = ProviderResult(
            provider_name="github",
            package="zapfast",
            success=True,
            message="Installed zapfast (.deb) via apt from GitHub release"
        )

        manager = ProviderManager(providers=[repo_prov, flatpak_prov, github_prov])
        orchestrator = InstallationOrchestrator(provider_manager=manager)

        result = orchestrator.install("zapfast")
        assert result.success
        assert result.provider_name == "github"
        assert "Installed zapfast" in result.message

