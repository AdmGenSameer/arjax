import os
import re
import shutil
import subprocess
from dataclasses import dataclass, field
from typing import Any, Dict, List, Set, Optional


def parse_size_bytes(val_str: str) -> int:
    val_str = val_str.strip()
    match = re.match(r"^([\d.]+)\s*([KMGTkmgt]?[iI]?[bB]?)", val_str)
    if not match:
        return 0
    try:
        num = float(match.group(1))
        unit = match.group(2).upper()
        if "G" in unit:
            return int(num * 1024 * 1024 * 1024)
        elif "M" in unit:
            return int(num * 1024 * 1024)
        elif "K" in unit:
            return int(num * 1024)
        return int(num)
    except ValueError:
        return 0


def format_size_bytes(num_bytes: int) -> str:
    if num_bytes >= 1024**3:
        return f"{num_bytes / (1024**3):.2f} GB"
    elif num_bytes >= 1024**2:
        return f"{num_bytes / (1024**2):.1f} MB"
    elif num_bytes >= 1024:
        return f"{num_bytes / 1024:.1f} KB"
    return f"{num_bytes} B"


@dataclass
class PackageNode:
    name: str
    is_explicit: bool = False
    dependencies: List[str] = field(default_factory=list)
    required_by: Set[str] = field(default_factory=set)
    provider: str = "repo"
    version: str = ""
    installed_size_bytes: int = 0


class OrphanDetector:
    """DAG-based dependency graph analyzer for detecting and cleaning up orphaned packages."""

    def __init__(self, engine: Optional[Any] = None):
        self.engine = engine

    def _query_arch_dag(self) -> Dict[str, PackageNode]:
        nodes: Dict[str, PackageNode] = {}
        pacman = shutil.which("pacman") or shutil.which("paru") or shutil.which("yay")
        if not pacman:
            return nodes

        # 1. Get explicit packages
        res_exp = subprocess.run([pacman, "-Qeq"], capture_output=True, text=True, check=False)
        explicit_pkgs = set(line.strip() for line in res_exp.stdout.split("\n") if line.strip())

        # 2. Get true orphans from pacman if available
        res_qdt = subprocess.run([pacman, "-Qdtq"], capture_output=True, text=True, check=False)
        pacman_orphans = set(line.strip() for line in res_qdt.stdout.split("\n") if line.strip())

        # 3. Get all installed packages with dependencies, required_by, & sizes
        res_all = subprocess.run([pacman, "-Qi"], capture_output=True, text=True, check=False)
        current_pkg: Optional[PackageNode] = None
        current_key: str = ""

        for line in res_all.stdout.split("\n"):
            if not line.strip():
                continue

            if ":" in line and not line.startswith(" "):
                parts = line.split(":", 1)
                key = parts[0].strip().lower()
                val = parts[1].strip()
                current_key = key

                if key == "name":
                    if current_pkg:
                        nodes[current_pkg.name] = current_pkg
                    current_pkg = PackageNode(
                        name=val,
                        is_explicit=(val in explicit_pkgs)
                    )
                elif current_pkg:
                    if key == "version":
                        current_pkg.version = val
                    elif key in ["installed size", "size"]:
                        current_pkg.installed_size_bytes = parse_size_bytes(val)
                    elif key in ["required by", "required_by"]:
                        if val and val.lower() != "none":
                            current_pkg.required_by.update(val.split())
                    elif key in ["depends on", "requires"]:
                        if val and val.lower() != "none":
                            deps = []
                            for dep_str in val.split():
                                clean = dep_str.split(">=")[0].split("<=")[0].split("=")[0].strip()
                                if clean and clean != "None":
                                    deps.append(clean)
                            current_pkg.dependencies = deps
            elif current_pkg and current_key:
                val = line.strip()
                if val and val.lower() != "none":
                    if current_key in ["required by", "required_by"]:
                        current_pkg.required_by.update(val.split())
                    elif current_key in ["depends on", "requires"]:
                        for dep_str in val.split():
                            clean = dep_str.split(">=")[0].split("<=")[0].split("=")[0].strip()
                            if clean and clean != "None":
                                current_pkg.dependencies.append(clean)

        if current_pkg:
            nodes[current_pkg.name] = current_pkg

        # Build in-degrees from dependencies
        for name, node in nodes.items():
            for dep in node.dependencies:
                if dep in nodes:
                    nodes[dep].required_by.add(name)

        # If pacman_orphans query succeeded, restrict non-explicit unreferenced nodes to pacman's true orphans
        if res_qdt.returncode == 0 or res_qdt.stdout.strip() == "":
            for name, node in nodes.items():
                if name not in pacman_orphans and not node.is_explicit:
                    # Package is protected because pacman does not mark it as a true orphan
                    node.required_by.add("system-protected")

        return nodes

    def _query_generic_dag(self) -> Dict[str, PackageNode]:
        nodes: Dict[str, PackageNode] = {}
        dpkg = shutil.which("dpkg-query")
        if dpkg:
            res = subprocess.run(
                [dpkg, "-W", "-f=${Package}\\t${Installed-Size}\\t${Status}\\n"],
                capture_output=True, text=True, check=False
            )
            for line in res.stdout.split("\n"):
                if "\t" in line and "installed" in line:
                    parts = line.split("\t")
                    pkg_name = parts[0].strip()
                    sz = parse_size_bytes(parts[1] + " KB") if len(parts) > 1 else 0
                    nodes[pkg_name] = PackageNode(name=pkg_name, is_explicit=False, installed_size_bytes=sz)
        return nodes

    def build_dag(self) -> Dict[str, PackageNode]:
        nodes = self._query_arch_dag()
        if not nodes:
            nodes = self._query_generic_dag()
        return nodes

    def find_orphans(self) -> List[Dict[str, Any]]:
        """Identify orphaned packages (leaf nodes installed as dependencies with 0 dependents)."""
        nodes = self.build_dag()
        orphans: List[Dict[str, Any]] = []

        if nodes:
            for name, node in nodes.items():
                if not node.is_explicit and len(node.required_by) == 0:
                    orphans.append({
                        "name": node.name,
                        "version": node.version,
                        "provider": node.provider,
                        "size_bytes": node.installed_size_bytes,
                        "size_str": format_size_bytes(node.installed_size_bytes),
                        "reason": "Unreferenced dependency (0 dependents)"
                    })
        else:
            pacman = shutil.which("pacman") or shutil.which("paru")
            if pacman:
                res = subprocess.run([pacman, "-Qdtq"], capture_output=True, text=True, check=False)
                for line in res.stdout.split("\n"):
                    pkg = line.strip()
                    if pkg:
                        orphans.append({
                            "name": pkg,
                            "version": "",
                            "provider": "repo",
                            "size_bytes": 0,
                            "size_str": "Unknown",
                            "reason": "Orphan package"
                        })

        return orphans

    def clean_orphans(self, orphan_names: List[str], engine: Any = None, dry_run: bool = False) -> Dict[str, Any]:
        """Safely remove specified orphan packages and calculate reclaimed disk space."""
        nodes = self.build_dag()
        total_size_bytes = sum(nodes[p].installed_size_bytes for p in orphan_names if p in nodes)
        total_size_str = format_size_bytes(total_size_bytes)

        if dry_run:
            return {
                "success": True,
                "dry_run": True,
                "orphans_removed": orphan_names,
                "count": len(orphan_names),
                "reclaimed_bytes": total_size_bytes,
                "reclaimed_size_str": total_size_str,
                "message": f"Dry run: would safely clean {len(orphan_names)} orphan package(s) and free {total_size_str} of disk space."
            }

        # 1. Try batch removal with elevated privileges if needed
        is_root = (os.geteuid() == 0) if hasattr(os, "geteuid") else False
        pacman = shutil.which("pacman")
        apt = shutil.which("apt-get")

        if pacman and orphan_names:
            cmd = ["pacman", "-Rns", "--noconfirm"] + orphan_names
            if not is_root and shutil.which("sudo"):
                cmd = ["sudo"] + cmd

            res = subprocess.run(cmd, check=False)
            if res.returncode == 0:
                return {
                    "success": True,
                    "dry_run": False,
                    "orphans_removed": orphan_names,
                    "failed": [],
                    "count": len(orphan_names),
                    "reclaimed_bytes": total_size_bytes,
                    "reclaimed_size_str": total_size_str
                }

        elif apt and orphan_names:
            cmd = ["apt-get", "remove", "-y"] + orphan_names
            if not is_root and shutil.which("sudo"):
                cmd = ["sudo"] + cmd

            res = subprocess.run(cmd, check=False)
            if res.returncode == 0:
                return {
                    "success": True,
                    "dry_run": False,
                    "orphans_removed": orphan_names,
                    "failed": [],
                    "count": len(orphan_names),
                    "reclaimed_bytes": total_size_bytes,
                    "reclaimed_size_str": total_size_str
                }

        # 2. Fallback to individual package removal loop
        eng = engine or self.engine
        removed = []
        failed = []

        for pkg in orphan_names:
            if eng:
                res = eng.remove(pkg)
                if isinstance(res, dict) and res.get("success", False):
                    removed.append(pkg)
                else:
                    failed.append(pkg)
            else:
                cmd = ["pacman", "-Rns", "--noconfirm", pkg] if pacman else ["apt-get", "remove", "-y", pkg]
                if not is_root and shutil.which("sudo"):
                    cmd = ["sudo"] + cmd
                res = subprocess.run(cmd, capture_output=True, check=False)
                if res.returncode == 0:
                    removed.append(pkg)
                else:
                    failed.append(pkg)

        freed_bytes = sum(nodes[p].installed_size_bytes for p in removed if p in nodes)

        return {
            "success": len(failed) == 0,
            "dry_run": False,
            "orphans_removed": removed,
            "failed": failed,
            "count": len(removed),
            "reclaimed_bytes": freed_bytes,
            "reclaimed_size_str": format_size_bytes(freed_bytes)
        }

