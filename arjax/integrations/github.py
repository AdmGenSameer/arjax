#!/usr/bin/env python3
"""
GitHub repository installation module for arjax
"""

import os
import sys
import shutil
import tempfile
import subprocess
import re
import urllib.request
import urllib.parse
import json
import platform
from typing import Optional, Dict, List, Tuple, Any
from pathlib import Path
from abc import ABC, abstractmethod

from arjax.config.logging import get_logger

logger = get_logger(__name__)

class ProjectTypeHandler(ABC):
    """Abstract base class for project type handlers"""

    @property
    @abstractmethod
    def name(self) -> str:
        """Name of the project type"""
        pass

    @property
    @abstractmethod
    def indicators(self) -> List[str]:
        """Files that indicate this project type"""
        pass

    @abstractmethod
    def can_handle(self, repo_path: Path) -> bool:
        """Check if this handler can handle the project"""
        pass

    @abstractmethod
    def build_and_install(self, repo_path: Path, temp_dir: Path) -> bool:
        """Build and install the project"""
        pass

    def log_command(self, command: List[str], description: str) -> None:
        """Log a command execution"""
        logger.info(f"{description}: {' '.join(command)}")
        print(f"  → {description}: {' '.join(command)}")

class PythonHandler(ProjectTypeHandler):
    """Handler for Python projects"""

    @property
    def name(self) -> str:
        return "Python"

    @property
    def indicators(self) -> List[str]:
        return ["setup.py", "pyproject.toml", "requirements.txt", "Pipfile"]

    def can_handle(self, repo_path: Path) -> bool:
        return any((repo_path / indicator).exists() for indicator in self.indicators)

    def build_and_install(self, repo_path: Path, temp_dir: Path) -> bool:
        try:
            # Change to repo directory
            os.chdir(repo_path)

            # Check for setup.py or pyproject.toml
            if (repo_path / "setup.py").exists():
                self.log_command(["pip", "install", "."], "Installing Python package")
                result = subprocess.run(["pip", "install", "."], capture_output=True, text=True)
            elif (repo_path / "pyproject.toml").exists():
                self.log_command(["pip", "install", "."], "Installing Python package (PEP 517)")
                result = subprocess.run(["pip", "install", "."], capture_output=True, text=True)
            else:
                print("  ⚠ No setup.py or pyproject.toml found, installing requirements if present")
                if (repo_path / "requirements.txt").exists():
                    self.log_command(["pip", "install", "-r", "requirements.txt"], "Installing requirements")
                    result = subprocess.run(["pip", "install", "-r", "requirements.txt"], capture_output=True, text=True)
                else:
                    print("  ✗ No installation method found for Python project")
                    return False

            if result.returncode == 0:
                print("  ✓ Python package installed successfully")
                return True
            else:
                print(f"  ✗ Python installation failed: {result.stderr}")
                return False

        except Exception as e:
            print(f"  ✗ Python installation error: {e}")
            return False

class NodeJSHandler(ProjectTypeHandler):
    """Handler for Node.js projects"""

    @property
    def name(self) -> str:
        return "Node.js"

    @property
    def indicators(self) -> List[str]:
        return ["package.json", "yarn.lock", "package-lock.json"]

    def can_handle(self, repo_path: Path) -> bool:
        return (repo_path / "package.json").exists()

    def build_and_install(self, repo_path: Path, temp_dir: Path) -> bool:
        try:
            os.chdir(repo_path)

            # Check if yarn or npm should be used
            use_yarn = (repo_path / "yarn.lock").exists()

            if use_yarn:
                # Install dependencies with yarn
                self.log_command(["yarn", "install"], "Installing dependencies with yarn")
                result = subprocess.run(["yarn", "install"], capture_output=True, text=True)
                if result.returncode != 0:
                    print(f"  ✗ Yarn install failed: {result.stderr}")
                    return False

                # Build if build script exists
                if self._has_build_script():
                    self.log_command(["yarn", "build"], "Building project with yarn")
                    result = subprocess.run(["yarn", "build"], capture_output=True, text=True)
                    if result.returncode != 0:
                        print(f"  ✗ Yarn build failed: {result.stderr}")
                        return False

                # Install globally if it's a CLI tool
                if self._is_cli_tool():
                    self.log_command(["yarn", "global", "add", "."], "Installing CLI tool globally")
                    result = subprocess.run(["yarn", "global", "add", "."], capture_output=True, text=True)
                else:
                    print("  ⚠ Not a CLI tool, skipping global installation")
                    return True

            else:
                # Use npm
                self.log_command(["npm", "install"], "Installing dependencies with npm")
                result = subprocess.run(["npm", "install"], capture_output=True, text=True)
                if result.returncode != 0:
                    print(f"  ✗ NPM install failed: {result.stderr}")
                    return False

                # Build if build script exists
                if self._has_build_script():
                    self.log_command(["npm", "run", "build"], "Building project with npm")
                    result = subprocess.run(["npm", "run", "build"], capture_output=True, text=True)
                    if result.returncode != 0:
                        print(f"  ✗ NPM build failed: {result.stderr}")
                        return False

                # Install globally if it's a CLI tool
                if self._is_cli_tool():
                    self.log_command(["npm", "install", "-g", "."], "Installing CLI tool globally")
                    result = subprocess.run(["npm", "install", "-g", "."], capture_output=True, text=True)
                else:
                    print("  ⚠ Not a CLI tool, skipping global installation")
                    return True

            if result.returncode == 0:
                print("  ✓ Node.js package installed successfully")
                return True
            else:
                print(f"  ✗ Node.js installation failed: {result.stderr}")
                return False

        except Exception as e:
            print(f"  ✗ Node.js installation error: {e}")
            return False

    def _has_build_script(self) -> bool:
        """Check if package.json has a build script"""
        try:
            import json
            with open("package.json", "r") as f:
                data = json.load(f)
                return "scripts" in data and "build" in data["scripts"]
        except:
            return False

    def _is_cli_tool(self) -> bool:
        """Check if this is a CLI tool by looking for bin field"""
        try:
            import json
            with open("package.json", "r") as f:
                data = json.load(f)
                return "bin" in data or ("name" in data and data["name"].startswith("@"))
        except:
            return False

class CMakeHandler(ProjectTypeHandler):
    """Handler for CMake projects"""

    @property
    def name(self) -> str:
        return "CMake"

    @property
    def indicators(self) -> List[str]:
        return ["CMakeLists.txt", "cmake"]

    def can_handle(self, repo_path: Path) -> bool:
        return (repo_path / "CMakeLists.txt").exists()

    def build_and_install(self, repo_path: Path, temp_dir: Path) -> bool:
        try:
            os.chdir(repo_path)

            # Create build directory
            build_dir = repo_path / "build"
            build_dir.mkdir(exist_ok=True)
            os.chdir(build_dir)

            # Configure with CMake
            self.log_command(["cmake", ".."], "Configuring with CMake")
            result = subprocess.run(["cmake", ".."], capture_output=True, text=True)
            if result.returncode != 0:
                print(f"  ✗ CMake configure failed: {result.stderr}")
                return False

            # Build
            self.log_command(["make", "-j$(nproc)"], "Building with make")
            result = subprocess.run(["make", f"-j{os.cpu_count() or 1}"], capture_output=True, text=True)
            if result.returncode != 0:
                print(f"  ✗ Make build failed: {result.stderr}")
                return False

            # Install
            from arjax.package_management.command_gen import build_privileged_command
            import shlex
            cmd_str = build_privileged_command("make install")
            cmd_list = shlex.split(cmd_str)
            self.log_command(cmd_list, "Installing with make install")
            result = subprocess.run(cmd_list, capture_output=True, text=True)
            if result.returncode == 0:
                print("  ✓ CMake project installed successfully")
                return True
            else:
                print(f"  ✗ Make install failed: {result.stderr}")
                return False

        except Exception as e:
            print(f"  ✗ CMake installation error: {e}")
            return False

class MakefileHandler(ProjectTypeHandler):
    """Handler for Makefile projects"""

    @property
    def name(self) -> str:
        return "Makefile"

    @property
    def indicators(self) -> List[str]:
        return ["Makefile", "makefile"]

    def can_handle(self, repo_path: Path) -> bool:
        return (repo_path / "Makefile").exists() or (repo_path / "makefile").exists()

    def build_and_install(self, repo_path: Path, temp_dir: Path) -> bool:
        try:
            os.chdir(repo_path)

            # Try make install first
            self.log_command(["make", "install"], "Installing with make")
            result = subprocess.run(["make", "install"], capture_output=True, text=True)

            if result.returncode == 0:
                print("  ✓ Makefile project installed successfully")
                return True
            else:
                # Try with elevated privileges if needed
                from arjax.package_management.command_gen import build_privileged_command
                import shlex
                cmd_str = build_privileged_command("make install")
                cmd_list = shlex.split(cmd_str)
                self.log_command(cmd_list, "Installing with make install")
                result = subprocess.run(cmd_list, capture_output=True, text=True)
                if result.returncode == 0:
                    print("  ✓ Makefile project installed successfully")
                    return True
                else:
                    print(f"  ✗ Make install failed: {result.stderr}")
                    return False

        except Exception as e:
            print(f"  ✗ Makefile installation error: {e}")
            return False

class GoHandler(ProjectTypeHandler):
    """Handler for Go projects"""

    @property
    def name(self) -> str:
        return "Go"

    @property
    def indicators(self) -> List[str]:
        return ["go.mod", "main.go", ".go"]

    def can_handle(self, repo_path: Path) -> bool:
        return (repo_path / "go.mod").exists() or any(f.suffix == ".go" for f in repo_path.glob("*.go"))

    def build_and_install(self, repo_path: Path, temp_dir: Path) -> bool:
        try:
            os.chdir(repo_path)

            # Install with go install
            self.log_command(["go", "install", "."], "Installing Go package")
            result = subprocess.run(["go", "install", "."], capture_output=True, text=True)

            if result.returncode == 0:
                print("  ✓ Go package installed successfully")
                return True
            else:
                print(f"  ✗ Go install failed: {result.stderr}")
                return False

        except Exception as e:
            print(f"  ✗ Go installation error: {e}")
            return False

class RustHandler(ProjectTypeHandler):
    """Handler for Rust projects"""

    @property
    def name(self) -> str:
        return "Rust"

    @property
    def indicators(self) -> List[str]:
        return ["Cargo.toml", "Cargo.lock"]

    def can_handle(self, repo_path: Path) -> bool:
        return (repo_path / "Cargo.toml").exists()

    def build_and_install(self, repo_path: Path, temp_dir: Path) -> bool:
        try:
            os.chdir(repo_path)

            # Install with cargo
            self.log_command(["cargo", "install", "--path", "."], "Installing Rust package")
            result = subprocess.run(["cargo", "install", "--path", "."], capture_output=True, text=True)

            if result.returncode == 0:
                print("  ✓ Rust package installed successfully")
                return True
            else:
                print(f"  ✗ Cargo install failed: {result.stderr}")
                return False

        except Exception as e:
            print(f"  ✗ Rust installation error: {e}")
            return False

class ProjectTypeRegistry:
    """Registry for project type handlers"""

    def __init__(self):
        self.handlers: List[ProjectTypeHandler] = [
            PythonHandler(),
            NodeJSHandler(),
            CMakeHandler(),
            MakefileHandler(),
            GoHandler(),
            RustHandler(),
        ]

    def detect_project_type(self, repo_path: Path) -> Optional[ProjectTypeHandler]:
        """Detect the project type and return appropriate handler"""
        for handler in self.handlers:
            if handler.can_handle(repo_path):
                return handler
        return None

    def get_supported_types(self) -> List[str]:
        """Get list of supported project types"""
        return [handler.name for handler in self.handlers]

def clone_repository(repo_url: str, temp_dir: Path) -> Optional[Path]:
    """Clone a GitHub repository to a temporary directory"""
    try:
        print(f"📥 Cloning repository: {repo_url}")

        # Use GitPython if available, otherwise use subprocess
        try:
            from git import Repo
            logger.info(f"Cloning {repo_url} to {temp_dir}")
            Repo.clone_from(repo_url, temp_dir)
            print("  ✓ Repository cloned successfully")
            return temp_dir
        except ImportError:
            # Fallback to subprocess
            logger.info(f"Cloning {repo_url} to {temp_dir} using subprocess")
            result = subprocess.run(["git", "clone", repo_url, str(temp_dir)], capture_output=True, text=True)
            if result.returncode == 0:
                print("  ✓ Repository cloned successfully")
                return temp_dir
            else:
                print(f"  ✗ Git clone failed: {result.stderr}")
                return None

    except Exception as e:
        print(f"  ✗ Repository cloning error: {e}")
        return None

def validate_github_url(url_or_repo: str) -> Optional[str]:
    """Validate and convert GitHub URL or user/repo format to full URL"""
    # Handle github:user/repo format
    if url_or_repo.startswith("github:"):
        repo = url_or_repo[7:]  # Remove "github:" prefix
        if "/" not in repo:
            print("  ✗ Invalid GitHub repo format. Use: github:user/repo")
            return None
        return f"https://github.com/{repo}.git"

    # Handle full GitHub URLs
    if url_or_repo.startswith("https://github.com/"):
        if not url_or_repo.endswith(".git"):
            url_or_repo += ".git"
        return url_or_repo

    # Handle git@github.com:user/repo.git format
    if url_or_repo.startswith("git@github.com:"):
        return url_or_repo

    # Handle bare user/repo format (e.g. crmne/zapfast)
    if "/" in url_or_repo and not url_or_repo.startswith("http") and not url_or_repo.startswith("/"):
        parts = url_or_repo.strip().split("/")
        if len(parts) == 2 and parts[0] and parts[1]:
            return f"https://github.com/{url_or_repo}.git"

    print("  ✗ Invalid GitHub URL format. Use: user/repo, github:user/repo or https://github.com/user/repo")
    return None

def install_from_github(repo_spec: str) -> bool:
    """Main function to install from GitHub repository"""
    print(f"\n🔧 Installing from GitHub: {repo_spec}")

    # Validate the repo specification
    repo_url = validate_github_url(repo_spec)
    if not repo_url:
        return False

    # Create temporary directory
    temp_dir = None
    try:
        temp_dir = Path(tempfile.mkdtemp(prefix="arjax-github-"))
        print(f"📁 Working in temporary directory: {temp_dir}")

        # Clone the repository
        repo_path = clone_repository(repo_url, temp_dir)
        if not repo_path:
            return False

        # Detect project type
        registry = ProjectTypeRegistry()
        handler = registry.detect_project_type(repo_path)

        if not handler:
            # List found files to help user
            files = list(repo_path.glob("*"))
            file_names = [f.name for f in files[:10]]  # Show first 10 files
            print(f"  ✗ Unsupported project type")
            print(f"  📄 Found files: {', '.join(file_names)}")
            if len(files) > 10:
                print(f"  ... and {len(files) - 10} more files")
            print(f"  💡 Supported types: {', '.join(registry.get_supported_types())}")
            print(f"  💡 Try manual installation or request support for this project type")
            return False

        print(f"  🔍 Detected project type: {handler.name}")

        # Build and install
        success = handler.build_and_install(repo_path, temp_dir)

        if success:
            print(f"🎉 Successfully installed {repo_spec}!")
            return True
        else:
            print(f"❌ Failed to install {repo_spec}")
            return False

    except Exception as e:
        print(f"  ✗ Unexpected error during installation: {e}")
        return False

    finally:
        # Clean up temporary directory
        if temp_dir and temp_dir.exists():
            print(f"🧹 Cleaning up temporary directory: {temp_dir}")
            try:
                shutil.rmtree(temp_dir)
                print("  ✓ Cleanup completed")
            except Exception as e:
                print(f"  ⚠ Cleanup warning: {e}")

def check_dependencies() -> Dict[str, bool]:
    """Check if required dependencies are available"""
    deps = {
        "git": False,
        "python": False,
        "node": False,
        "npm": False,
        "yarn": False,
        "cmake": False,
        "make": False,
        "go": False,
        "cargo": False,
    }

    # Check each dependency
    for dep in deps.keys():
        try:
            result = subprocess.run([dep, "--version"], capture_output=True, timeout=5)
            deps[dep] = result.returncode == 0
        except (subprocess.TimeoutExpired, FileNotFoundError, subprocess.SubprocessError):
            deps[dep] = False

    return deps


def search_github_repositories(query: str, limit: int = 5) -> List[Dict[str, Any]]:
    """Search GitHub repositories using the GitHub API.
    
    Args:
        query: Repository search query
        limit: Max results to return
        
    Returns:
        List of repository information dicts
    """
    clean_query = query.strip()
    encoded = urllib.parse.quote_plus(clean_query)
    url = f"https://api.github.com/search/repositories?q={encoded}&sort=stars&order=desc"
    req = urllib.request.Request(
        url,
        headers={
            "User-Agent": "arjax/2.0 (Linux; cross-distro package manager)",
            "Accept": "application/vnd.github.v3+json",
        }
    )
    try:
        with urllib.request.urlopen(req, timeout=12) as response:
            data = json.loads(response.read().decode("utf-8"))
            items = data.get("items", [])
            results = []
            for item in items[:limit]:
                results.append({
                    "name": item.get("name", ""),
                    "full_name": item.get("full_name", ""),
                    "description": item.get("description", ""),
                    "stars": item.get("stargazers_count", 0),
                    "html_url": item.get("html_url", ""),
                    "default_branch": item.get("default_branch", "main"),
                })
            return results
    except Exception as e:
        logger.debug(f"GitHub repository search failed for '{query}': {e}")
        return []


def get_latest_github_release(repo_full_name: str) -> Optional[Dict[str, Any]]:
    """Fetch latest release info and assets for a GitHub repository.
    
    Args:
        repo_full_name: Repository in 'owner/repo' format
        
    Returns:
        Release dictionary with tag_name, name, and assets, or None
    """
    headers = {
        "User-Agent": "arjax/2.0 (Linux; cross-distro package manager)",
        "Accept": "application/vnd.github.v3+json",
    }
    # First try /releases/latest
    latest_url = f"https://api.github.com/repos/{repo_full_name}/releases/latest"
    try:
        req = urllib.request.Request(latest_url, headers=headers)
        with urllib.request.urlopen(req, timeout=12) as response:
            data = json.loads(response.read().decode("utf-8"))
            return _format_release_data(data)
    except urllib.error.HTTPError as http_err:
        if http_err.code != 404:
            logger.debug(f"GitHub release fetch HTTP error {http_err.code} for {repo_full_name}")
            return None
    except Exception as e:
        logger.debug(f"GitHub latest release fetch failed for {repo_full_name}: {e}")
        return None

    # Fallback to /releases (first non-draft)
    releases_url = f"https://api.github.com/repos/{repo_full_name}/releases?per_page=5"
    try:
        req = urllib.request.Request(releases_url, headers=headers)
        with urllib.request.urlopen(req, timeout=12) as response:
            releases = json.loads(response.read().decode("utf-8"))
            for rel in releases:
                if not rel.get("draft", False):
                    return _format_release_data(rel)
    except Exception as e:
        logger.debug(f"GitHub releases fallback list failed for {repo_full_name}: {e}")
        return None

    return None


def _format_release_data(data: Dict[str, Any]) -> Dict[str, Any]:
    assets = []
    for a in data.get("assets", []):
        assets.append({
            "name": a.get("name", ""),
            "browser_download_url": a.get("browser_download_url", ""),
            "size": a.get("size", 0),
            "content_type": a.get("content_type", ""),
        })
    return {
        "tag_name": data.get("tag_name", ""),
        "name": data.get("name", ""),
        "body": data.get("body", ""),
        "assets": assets,
    }


def select_best_github_asset(
    assets: List[Dict[str, Any]],
    distro_family: str,
    machine_arch: Optional[str] = None,
    preferred_format: Optional[str] = None
) -> Optional[Dict[str, Any]]:
    """Select the most suitable release asset for the current OS and architecture.
    
    Args:
        assets: List of release asset dictionaries
        distro_family: Distribution family ('debian', 'arch', 'fedora', 'suse')
        machine_arch: Machine architecture (defaults to platform.machine())
        preferred_format: Optional forced format ('appimage', 'deb', 'rpm')
        
    Returns:
        Best matching asset dictionary or None
    """
    if not assets:
        return None

    arch = (machine_arch or platform.machine()).lower()
    is_x86_64 = arch in ("x86_64", "amd64", "x64")
    is_arm64 = arch in ("aarch64", "arm64", "armv8")

    def score_asset(asset: Dict[str, Any]) -> int:
        name = asset.get("name", "").lower()
        if not name:
            return -1000

        # Exclude non-Linux files
        if any(bad in name for bad in [".exe", ".msi", ".dmg", ".pkg", "windows", "win32", "win64", "win-x64", "darwin", "macos", "apple"]):
            return -1000
        # Exclude checksums and signatures
        if any(name.endswith(ext) for ext in [".sha256", ".sha512", ".md5", ".sig", ".asc", ".blockmap", ".txt", ".sha1"]):
            return -1000
        # Exclude source/packaging-only tars
        if "packaging" in name or "checksum" in name:
            return -1000

        # Architecture filtering
        if is_x86_64:
            if any(a in name for a in ["arm64", "aarch64", "armv7", "armv6", "i386", "i686"]):
                return -1000
        elif is_arm64:
            if any(a in name for a in ["x86_64", "amd64", "x64", "i386", "i686"]):
                return -1000

        score = 0

        # Architecture affinity
        if is_x86_64 and any(a in name for a in ["x86_64", "amd64", "x64"]):
            score += 50
        elif is_arm64 and any(a in name for a in ["aarch64", "arm64"]):
            score += 50

        # Explicit user preference boost
        if preferred_format == "appimage" and name.endswith(".appimage"):
            score += 300
        elif preferred_format == "deb" and name.endswith(".deb"):
            score += 300
        elif preferred_format == "rpm" and name.endswith(".rpm"):
            score += 300

        # Distro family preferences
        if distro_family == "debian":
            if name.endswith(".deb"):
                score += 150
            elif name.endswith(".appimage"):
                score += 100
            elif name.endswith(".flatpak"):
                score += 80
            elif name.endswith((".tar.gz", ".tar.xz", ".tgz")) and "linux" in name:
                score += 40
        elif distro_family == "fedora":
            if name.endswith(".rpm"):
                score += 150
            elif name.endswith(".appimage"):
                score += 100
            elif name.endswith((".tar.gz", ".tar.xz", ".tgz")) and "linux" in name:
                score += 40
        elif distro_family == "arch":
            if name.endswith(".appimage"):
                score += 120
            elif name.endswith((".tar.gz", ".tar.xz", ".tgz")) and "linux" in name:
                score += 60
            elif name.endswith(".deb"):
                score += 20
        else:
            if name.endswith(".appimage"):
                score += 120
            elif name.endswith(".deb") or name.endswith(".rpm"):
                score += 80
            elif name.endswith((".tar.gz", ".tar.xz", ".tgz")) and "linux" in name:
                score += 50

        return score

    scored = [(score_asset(a), a) for a in assets]
    valid = [item for item in scored if item[0] > 0]
    if not valid:
        return None

    valid.sort(key=lambda x: x[0], reverse=True)
    return valid[0][1]
