"""Provider implementations for the installation engine."""

from __future__ import annotations

import json
import os
import platform
import re
import shlex
import shutil
import subprocess
import tarfile
import tempfile
import zipfile
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Optional, List, Dict, Any
from urllib.request import urlopen, Request

import distro

from arjax.config.base import DISTRO_MAP
from arjax.config.logging import get_logger
from arjax.core.exceptions import CommandGenerationError, PackageManagerNotFound, ValidationError
from arjax.installation.models import ProviderConfig, ProviderResult, Recipe
from arjax.installation.recipes import RecipeStore
from arjax.integrations.github import (
    install_from_github,
    search_github_repositories,
    get_latest_github_release,
    select_best_github_asset,
    validate_github_url,
)
from arjax.package_management.command_gen import generate_command

logger = get_logger(__name__)


_PROVIDER_ORDER = {
    "repository": 10,
    "vendor": 20,
    "flatpak": 40,
    "github": 45,
    "snap": 50,
    "appimage": 60,
}


def detect_distro_family() -> str:
    """Return a coarse distro family key using existing distro detection data."""
    detected = distro.id().lower().strip()
    return DISTRO_MAP.get(detected, detected)


def default_package_manager() -> str:
    """Return the default native package manager name for the current distro."""
    family = detect_distro_family()
    if family == "arch":
        from arjax.package_management.command_gen import check_command_availability
        if check_command_availability("paru"):
            return "paru"
        return "pacman"
    if family == "debian":
        return "apt"
    if family == "fedora":
        return "dnf"
    if family == "suse":
        return "zypper"
    return "pacman"


def is_package_in_repo(package_name: str, package_manager: str) -> bool:
    """Check whether a package exists in the native repository."""
    try:
        if package_manager == "apt":
            res = subprocess.run(["apt-cache", "show", package_name], capture_output=True, timeout=5)
            return res.returncode == 0
        elif package_manager in ("pacman", "paru"):
            tool = "paru" if shutil.which("paru") else "pacman"
            res = subprocess.run([tool, "-Si", package_name], capture_output=True, timeout=5)
            return res.returncode == 0
        elif package_manager == "dnf":
            res = subprocess.run(["dnf", "info", package_name], capture_output=True, timeout=8)
            return res.returncode == 0
        elif package_manager == "zypper":
            res = subprocess.run(["zypper", "info", package_name], capture_output=True, timeout=8)
            return res.returncode == 0
    except Exception:
        pass
    return True


def create_desktop_entry(
    package_name: str,
    exec_path: Path,
    display_name: Optional[str] = None,
    icon: Optional[str] = None,
    terminal: bool = False,
    comment: Optional[str] = None,
) -> Path:
    """Create an XDG .desktop file for system launcher / desktop integration."""
    apps_dir = Path.home() / ".local" / "share" / "applications"
    apps_dir.mkdir(parents=True, exist_ok=True)

    clean_name = package_name.lower().replace(" ", "-")
    desktop_file = apps_dir / f"{clean_name}.desktop"
    name = display_name or package_name.title()
    icon_name = icon or clean_name

    content = [
        "[Desktop Entry]",
        "Version=1.0",
        "Type=Application",
        f"Name={name}",
        f"Exec={exec_path} %U",
        f"Icon={icon_name}",
        f"Terminal={'true' if terminal else 'false'}",
        "Categories=Utility;Application;",
    ]
    if comment:
        content.append(f"Comment={comment}")

    desktop_file.write_text("\n".join(content) + "\n", encoding="utf-8")

    try:
        if shutil.which("update-desktop-database"):
            subprocess.run(["update-desktop-database", str(apps_dir)], capture_output=True, timeout=5)
    except Exception:
        pass

    return desktop_file


def is_verbose() -> bool:
    """Return True if console logging level is at DEBUG (verbose mode)."""
    import logging
    for handler in logging.getLogger().handlers:
        if isinstance(handler, logging.StreamHandler) and not isinstance(handler, logging.FileHandler):
            return handler.level == logging.DEBUG
    return False


class Provider(ABC):
    """Base class for all installation providers."""

    name: str

    @property
    def priority(self) -> int:
        return _PROVIDER_ORDER.get(self.name, 100)

    @abstractmethod
    def supports(self, package: str, recipe: Optional[Recipe]) -> bool:
        """Return True when this provider should be considered."""

    @abstractmethod
    def install(self, package: str, recipe: Optional[Recipe]) -> ProviderResult:
        """Attempt an installation and return a structured result."""

    def _skip(self, package: str, message: str) -> ProviderResult:
        return ProviderResult(
            provider_name=self.name,
            package=package,
            success=False,
            skipped=True,
            message=message,
        )


class RepositoryProvider(Provider):
    """Install directly from the native repository using the current distro PM."""

    name = "repository"

    def supports(self, package: str, recipe: Optional[Recipe]) -> bool:
        return True

    def install(self, package: str, recipe: Optional[Recipe]) -> ProviderResult:
        provider_config = recipe.get_provider(self.name) if recipe else None
        package_name = (provider_config.package if provider_config and provider_config.package else package).strip()
        package_manager = (provider_config.package_manager if provider_config and provider_config.package_manager else default_package_manager()).strip()

        # If no recipe, verify package exists in native repository so we can cleanly fallback
        if not recipe and not is_package_in_repo(package_name, package_manager):
            return self._skip(package, f"Package '{package_name}' not found in {package_manager} repositories")

        try:
            command = generate_command(package_name, package_manager)
            if not command:
                return self._skip(package, f"No install command could be generated for {package_name}")

            if is_verbose():
                from rich.console import Console
                Console().print(f"[bold dim]$ {command}[/bold dim]")

            result = subprocess.run(shlex.split(command), capture_output=True, text=True)
            if result.returncode == 0:
                return ProviderResult(
                    provider_name=self.name,
                    package=package,
                    success=True,
                    message=f"Installed {package_name} via {package_manager}",
                    details={"command": command, "stdout": result.stdout.strip()},
                )

            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"Repository install failed for {package_name}",
                error=(result.stderr or result.stdout or "Unknown error").strip(),
                details={"command": command, "returncode": result.returncode},
            )
        except (PackageManagerNotFound, ValidationError, CommandGenerationError) as exc:
            return self._skip(package, str(exc))
        except Exception as exc:
            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"Repository provider failed for {package_name}",
                error=str(exc),
            )


class VendorProvider(Provider):
    """Install from a vendor-maintained package mapping stored in a recipe."""

    name = "vendor"

    def supports(self, package: str, recipe: Optional[Recipe]) -> bool:
        return bool(recipe and recipe.has_provider(self.name))

    def install(self, package: str, recipe: Optional[Recipe]) -> ProviderResult:
        if not recipe:
            return self._skip(package, "No recipe available for vendor install")

        provider_config = recipe.get_provider(self.name)
        if not provider_config:
            return self._skip(package, "Recipe does not define a vendor provider")

        package_name = (provider_config.package or package).strip()
        package_manager = (provider_config.package_manager or default_package_manager()).strip()

        try:
            command = generate_command(package_name, package_manager)
            if not command:
                return self._skip(package, f"No install command could be generated for {package_name}")

            if is_verbose():
                from rich.console import Console
                Console().print(f"[bold dim]$ {command}[/bold dim]")

            result = subprocess.run(shlex.split(command), capture_output=True, text=True)
            if result.returncode == 0:
                return ProviderResult(
                    provider_name=self.name,
                    package=package,
                    success=True,
                    message=f"Installed {package_name} from vendor mapping via {package_manager}",
                    details={"command": command},
                )

            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"Vendor install failed for {package_name}",
                error=(result.stderr or result.stdout or "Unknown error").strip(),
                details={"command": command, "returncode": result.returncode},
            )
        except (PackageManagerNotFound, ValidationError, CommandGenerationError) as exc:
            return self._skip(package, str(exc))
        except Exception as exc:
            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"Vendor provider failed for {package_name}",
                error=str(exc),
            )


class GithubReleaseProvider(Provider):
    """Install from explicitly declared GitHub release metadata or dynamic GitHub discovery."""

    name = "github"

    def supports(self, package: str, recipe: Optional[Recipe]) -> bool:
        return True

    def install(self, package: str, recipe: Optional[Recipe]) -> ProviderResult:
        if recipe and recipe.has_provider(self.name):
            provider_config = recipe.get_provider(self.name)
            if provider_config:
                if provider_config.github_install_type == "source" and provider_config.github_repo:
                    ok = install_from_github(provider_config.github_repo)
                    if ok:
                        return ProviderResult(
                            provider_name=self.name,
                            package=package,
                            success=True,
                            message=f"Installed {package} from GitHub source recipe",
                            details={"repo": provider_config.github_repo},
                        )
                    return ProviderResult(
                        provider_name=self.name,
                        package=package,
                        success=False,
                        message=f"GitHub source install failed for {package}",
                        details={"repo": provider_config.github_repo},
                    )

                if provider_config.github_download_url:
                    return self._install_downloaded_binary(package, provider_config, provider_config.github_download_url)

                if provider_config.github_repo:
                    return self._resolve_and_install_repo(package, provider_config.github_repo, provider_config)

        # Dynamic automated GitHub resolution
        return self._resolve_and_install_package(package)

    def _resolve_and_install_package(self, package: str, preferred_format: Optional[str] = None) -> ProviderResult:
        """Dynamically search GitHub for a matching repository, release assets, and install."""
        repo_name = None
        clean_pkg = package.strip()

        if clean_pkg.startswith("github:"):
            repo_name = clean_pkg[7:]
        elif "github.com/" in clean_pkg:
            match = re.search(r"github\.com/([^/]+/[^/]+?)(?:\.git|/)?$", clean_pkg)
            if match:
                repo_name = match.group(1)
        elif "/" in clean_pkg and not clean_pkg.startswith("/"):
            parts = clean_pkg.split("/")
            if len(parts) == 2:
                repo_name = clean_pkg

        if not repo_name:
            repos = search_github_repositories(clean_pkg, limit=5)
            if not repos:
                return self._skip(package, f"No matching GitHub repository found for '{package}'")

            norm_pkg = clean_pkg.lower().replace(" ", "").replace("-", "").replace("_", "")
            exact_match = None
            for r in repos:
                norm_repo_name = r["name"].lower().replace(" ", "").replace("-", "").replace("_", "")
                if norm_pkg == norm_repo_name:
                    exact_match = r
                    break

            best_repo = exact_match or repos[0]
            repo_name = best_repo["full_name"]

        return self._resolve_and_install_repo(package, repo_name, preferred_format=preferred_format)

    def _resolve_and_install_repo(
        self,
        package: str,
        repo_full_name: str,
        provider_config: Optional[ProviderConfig] = None,
        preferred_format: Optional[str] = None
    ) -> ProviderResult:
        """Inspect latest release assets for repo or compile from source as fallback."""
        cfg = provider_config or ProviderConfig(appimage_name=package)
        release = get_latest_github_release(repo_full_name)

        if release and release.get("assets"):
            family = detect_distro_family()
            asset = select_best_github_asset(release["assets"], family, preferred_format=preferred_format)
            if asset and asset.get("browser_download_url"):
                download_url = asset["browser_download_url"]
                return self._install_downloaded_binary(package, cfg, download_url, repo=repo_full_name)

        # Fallback: if no release binary assets found, compile from source
        logger.info("No suitable binary asset found for %s; attempting source build", repo_full_name)
        ok = install_from_github(repo_full_name)
        if ok:
            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=True,
                message=f"Compiled and installed {package} from GitHub source ({repo_full_name})",
                details={"repo": repo_full_name},
            )

        return ProviderResult(
            provider_name=self.name,
            package=package,
            success=False,
            message=f"GitHub install failed for {package}: no suitable binary or successful source build from {repo_full_name}",
            details={"repo": repo_full_name},
        )

    def _install_downloaded_binary(
        self,
        package: str,
        provider_config: ProviderConfig,
        download_url: str,
        repo: Optional[str] = None
    ) -> ProviderResult:
        suffix = Path(download_url).name
        temp_dir = Path(tempfile.mkdtemp(prefix="arjax-github-"))
        try:
            archive_path = temp_dir / suffix

            if is_verbose():
                from rich.console import Console
                Console().print(f"[bold dim]$ (downloading {download_url})[/bold dim]")

            req = Request(download_url, headers={"User-Agent": "arjax/2.0 (Linux)"})
            with urlopen(req) as response:
                content_length = response.info().get('Content-Length')
                total_size = int(content_length) if content_length else None

                if total_size:
                    from rich.progress import Progress, BarColumn, TextColumn, TimeRemainingColumn, DownloadColumn
                    from rich.console import Console

                    with Progress(
                        TextColumn("[bold blue]{task.description}"),
                        BarColumn(),
                        DownloadColumn(),
                        TimeRemainingColumn(),
                        console=Console()
                    ) as progress:
                        task = progress.add_task(f"Downloading {suffix}...", total=total_size)
                        with open(archive_path, 'wb') as f:
                            while True:
                                chunk = response.read(16384)
                                if not chunk:
                                    break
                                f.write(chunk)
                                progress.update(task, advance=len(chunk))
                else:
                    archive_path.write_bytes(response.read())

            lower_name = archive_path.name.lower()

            # 1. Handle .deb package (Pop!_OS / Debian / Ubuntu)
            if lower_name.endswith(".deb"):
                from arjax.package_management.command_gen import build_privileged_command
                cmd_str = build_privileged_command(f"apt install -y {archive_path}")
                if is_verbose():
                    from rich.console import Console
                    Console().print(f"[bold dim]$ {cmd_str}[/bold dim]")
                result = subprocess.run(shlex.split(cmd_str), capture_output=True, text=True)
                if result.returncode == 0:
                    return ProviderResult(
                        provider_name=self.name,
                        package=package,
                        success=True,
                        message=f"Installed {package} (.deb) via apt from GitHub release",
                        details={"download_url": download_url, "path": str(archive_path), "repo": repo},
                    )
                return ProviderResult(
                    provider_name=self.name,
                    package=package,
                    success=False,
                    message=f"APT failed to install deb package for {package}",
                    error=(result.stderr or result.stdout or "Unknown error").strip(),
                    details={"download_url": download_url, "repo": repo},
                )

            # 2. Handle .AppImage
            if lower_name.endswith(".appimage"):
                return self._install_appimage(archive_path, provider_config, package)

            # 3. Handle .tar.gz / .tar.xz / .zip archive containing executable
            if any(lower_name.endswith(ext) for ext in [".tar.gz", ".tar.xz", ".tgz", ".zip"]):
                return self._install_archive(archive_path, provider_config, package)

            # 4. Fallback binary
            return self._install_appimage(archive_path, provider_config, package)

        except Exception as exc:
            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"GitHub release download failed for {package}",
                error=str(exc),
            )
        finally:
            shutil.rmtree(temp_dir, ignore_errors=True)

    def _install_appimage(self, archive_path: Path, provider_config: ProviderConfig, package: str) -> ProviderResult:
        raw_name = provider_config.appimage_name or package or archive_path.stem
        target_name = raw_name[:-9] if raw_name.lower().endswith(".appimage") else raw_name
        target_dir = Path.home() / ".local" / "bin"
        target_dir.mkdir(parents=True, exist_ok=True)
        target_path = target_dir / target_name
        shutil.copy2(archive_path, target_path)
        target_path.chmod(target_path.stat().st_mode | 0o111)

        try:
            create_desktop_entry(
                package_name=target_name,
                exec_path=target_path,
                display_name=target_name.title(),
                comment=f"{target_name} installed via Arjax",
            )
        except Exception as e:
            logger.debug(f"Failed to create desktop entry for {target_name}: {e}")

        return ProviderResult(
            provider_name=self.name,
            package=package,
            success=True,
            message=f"Installed AppImage from GitHub release to {target_path}",
            details={"path": str(target_path)},
        )

    def _install_archive(self, archive_path: Path, provider_config: ProviderConfig, package: str) -> ProviderResult:
        extract_dir = archive_path.parent / "extracted"
        extract_dir.mkdir(exist_ok=True)
        if archive_path.name.lower().endswith(".zip"):
            with zipfile.ZipFile(archive_path, 'r') as zip_ref:
                zip_ref.extractall(extract_dir)
        else:
            with tarfile.open(archive_path, 'r:*') as tar_ref:
                tar_ref.extractall(extract_dir)

        candidates = []
        for p in extract_dir.rglob("*"):
            if p.is_file() and not p.name.endswith((".txt", ".md", ".png", ".svg", ".desktop", ".so", ".h", ".c")):
                candidates.append(p)

        target_bin = None
        clean_pkg = package.lower().replace(" ", "").replace("-", "")
        for c in candidates:
            if clean_pkg in c.name.lower():
                target_bin = c
                break
        if not target_bin and candidates:
            target_bin = candidates[0]

        if not target_bin:
            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"No executable found in archive for {package}",
            )

        raw_name = provider_config.appimage_name or package
        target_name = raw_name[:-9] if raw_name.lower().endswith(".appimage") else raw_name
        target_dir = Path.home() / ".local" / "bin"
        target_dir.mkdir(parents=True, exist_ok=True)
        target_path = target_dir / target_name
        shutil.copy2(target_bin, target_path)
        target_path.chmod(target_path.stat().st_mode | 0o111)

        create_desktop_entry(
            package_name=target_name,
            exec_path=target_path,
            display_name=target_name.title(),
        )

        return ProviderResult(
            provider_name=self.name,
            package=package,
            success=True,
            message=f"Installed binary from GitHub release archive to {target_path}",
            details={"path": str(target_path)},
        )


class FlatpakProvider(Provider):
    """Install via Flatpak when the recipe declares a Flatpak app ID or dynamically via Flathub."""

    name = "flatpak"

    def supports(self, package: str, recipe: Optional[Recipe]) -> bool:
        if recipe and recipe.has_provider(self.name):
            return True
        from arjax.package_management.command_gen import check_command_availability
        return check_command_availability("flatpak")

    def install(self, package: str, recipe: Optional[Recipe]) -> ProviderResult:
        provider_config = recipe.get_provider(self.name) if recipe else None
        flatpak_id = None
        if provider_config:
            flatpak_id = (provider_config.flatpak_id or provider_config.package or package).strip()
        else:
            # Dynamic search via Flathub
            flatpak_id = self._find_flatpak_app_id(package)
            if not flatpak_id:
                return self._skip(package, f"No Flatpak app found matching '{package}'")

        try:
            command = generate_command(flatpak_id, "flatpak")
            if not command:
                return self._skip(package, f"No Flatpak command could be generated for {flatpak_id}")

            if is_verbose():
                from rich.console import Console
                Console().print(f"[bold dim]$ {command}[/bold dim]")

            result = subprocess.run(shlex.split(command), capture_output=True, text=True)
            if result.returncode == 0:
                return ProviderResult(
                    provider_name=self.name,
                    package=package,
                    success=True,
                    message=f"Installed {flatpak_id} via Flatpak",
                    details={"command": command},
                )

            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"Flatpak install failed for {flatpak_id}",
                error=(result.stderr or result.stdout or "Unknown error").strip(),
                details={"command": command, "returncode": result.returncode},
            )
        except (PackageManagerNotFound, ValidationError, CommandGenerationError) as exc:
            return self._skip(package, str(exc))
        except Exception as exc:
            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"Flatpak provider failed for {flatpak_id}",
                error=str(exc),
            )

    def _find_flatpak_app_id(self, query: str) -> Optional[str]:
        try:
            from arjax.search.flatpak import search_flatpak
            results = search_flatpak(query)
            if results:
                clean_query = query.lower().strip()
                for app_id, desc, _ in results:
                    parts = app_id.lower().split(".")
                    if clean_query in parts or clean_query == app_id.lower():
                        return app_id
                return results[0][0]
        except Exception as e:
            logger.debug(f"Flatpak dynamic search failed: {e}")
        return None


class SnapProvider(Provider):
    """Install via Snap when the recipe declares a snap name."""

    name = "snap"

    def supports(self, package: str, recipe: Optional[Recipe]) -> bool:
        return bool(recipe and recipe.has_provider(self.name))

    def install(self, package: str, recipe: Optional[Recipe]) -> ProviderResult:
        if not recipe:
            return self._skip(package, "No recipe available for Snap install")

        provider_config = recipe.get_provider(self.name)
        if not provider_config:
            return self._skip(package, "Recipe does not define a Snap provider")

        snap_name = (provider_config.snap_id or provider_config.package or package).strip()
        try:
            command = generate_command(snap_name, "snap")
            if not command:
                return self._skip(package, f"No Snap command could be generated for {snap_name}")

            if is_verbose():
                from rich.console import Console
                Console().print(f"[bold dim]$ {command}[/bold dim]")

            result = subprocess.run(shlex.split(command), capture_output=True, text=True)
            if result.returncode == 0:
                return ProviderResult(
                    provider_name=self.name,
                    package=package,
                    success=True,
                    message=f"Installed {snap_name} via Snap",
                    details={"command": command},
                )

            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"Snap install failed for {snap_name}",
                error=(result.stderr or result.stdout or "Unknown error").strip(),
                details={"command": command, "returncode": result.returncode},
            )
        except (PackageManagerNotFound, ValidationError, CommandGenerationError) as exc:
            return self._skip(package, str(exc))
        except Exception as exc:
            return ProviderResult(
                provider_name=self.name,
                package=package,
                success=False,
                message=f"Snap provider failed for {snap_name}",
                error=str(exc),
            )


class AppImageProvider(Provider):
    """Install AppImage assets from recipes or dynamic GitHub releases."""

    name = "appimage"

    def supports(self, package: str, recipe: Optional[Recipe]) -> bool:
        return True

    def install(self, package: str, recipe: Optional[Recipe]) -> ProviderResult:
        if recipe and recipe.has_provider(self.name):
            provider_config = recipe.get_provider(self.name)
            if provider_config and provider_config.appimage_url:
                gh_prov = GithubReleaseProvider()
                gh_prov.name = self.name
                return gh_prov._install_downloaded_binary(
                    package,
                    provider_config,
                    provider_config.appimage_url,
                )

        # Dynamic automated GitHub resolution with preference for AppImage format
        gh_prov = GithubReleaseProvider()
        gh_prov.name = self.name
        return gh_prov._resolve_and_install_package(package, preferred_format="appimage")


def default_providers() -> list[Provider]:
    """Return providers in the requested priority order."""
    return [
        RepositoryProvider(),
        VendorProvider(),
        GithubReleaseProvider(),
        FlatpakProvider(),
        SnapProvider(),
        AppImageProvider(),
    ]


class ProviderManager:
    """Tiny provider loop that delegates to the first provider that works."""

    def __init__(self, providers: Optional[list[Provider]] = None, recipes: Optional[RecipeStore] = None):
        self.providers = sorted(providers or default_providers(), key=lambda provider: provider.priority)
        self.recipes = recipes or RecipeStore()

    def list_providers(self) -> list[str]:
        return [provider.name for provider in self.providers]

    def install(self, package: str, provider_hint: Optional[str] = None) -> ProviderResult:
        recipe = self.recipes.find(package)

        for provider in self.providers:
            if provider_hint and provider.name != provider_hint.lower().strip():
                continue

            if not provider.supports(package, recipe):
                continue

            result = provider.install(package, recipe)
            if result.success:
                return result

            if not result.skipped:
                return result

        return ProviderResult(
            provider_name="manager",
            package=package,
            success=False,
            message=f"No provider could install {package}",
            details={"providers": self.list_providers(), "recipe_found": bool(recipe)},
        )
