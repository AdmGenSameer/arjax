"""System Statistics & Health Diagnostics Engine for Arjax."""

from dataclasses import dataclass, field
import os
from pathlib import Path
import shutil
import subprocess
from typing import Any, Dict, List, Optional
from arjax.core.orphans import OrphanDetector


@dataclass
class DiskMetric:
    path: str
    total_bytes: int
    used_bytes: int
    free_bytes: int
    percent_used: float

    def formatted_used(self) -> str:
        return f"{self.used_bytes / (1024**3):.2f} GB"

    def formatted_total(self) -> str:
        return f"{self.total_bytes / (1024**3):.2f} GB"


@dataclass
class HealthCheckAlert:
    category: str
    severity: str  # "info", "warning", "critical"
    message: str
    suggestion: str = ""


@dataclass
class SystemStatistics:
    root_disk: Optional[DiskMetric] = None
    arjax_cache_bytes: int = 0
    pkg_cache_bytes: int = 0
    package_counts: Dict[str, int] = field(default_factory=dict)
    pending_updates: int = 0
    orphans_count: int = 0
    failed_services_count: int = 0
    health_alerts: List[HealthCheckAlert] = field(default_factory=list)


def get_dir_size(path: Path) -> int:
    """Calculate total size of directory in bytes safely."""
    total = 0
    if not path.exists():
        return 0
    try:
        for p in path.rglob("*"):
            if p.is_file() and not p.is_symlink():
                total += p.stat().st_size
    except Exception:
        pass
    return total


def gather_system_statistics() -> SystemStatistics:
    """Gather comprehensive system disk usage, package counts, updates, and health alerts."""
    stats = SystemStatistics()

    # 1. Disk usage metric for /
    try:
        usage = shutil.disk_usage("/")
        stats.root_disk = DiskMetric(
            path="/",
            total_bytes=usage.total,
            used_bytes=usage.used,
            free_bytes=usage.free,
            percent_used=(usage.used / usage.total) * 100 if usage.total > 0 else 0
        )
        if stats.root_disk.percent_used > 85.0:
            stats.health_alerts.append(HealthCheckAlert(
                category="Storage",
                severity="critical" if stats.root_disk.percent_used > 92.0 else "warning",
                message=f"Root disk partition '/' is {stats.root_disk.percent_used:.1f}% full.",
                suggestion="Run 'arjax cleanup cache' or remove unused packages."
            ))
    except Exception:
        pass

    # Cache sizes
    stats.arjax_cache_bytes = get_dir_size(Path.home() / ".cache" / "arjax")
    stats.pkg_cache_bytes = get_dir_size(Path("/var/cache/pacman/pkg"))

    # 2. Package Counts per Provider
    # Arch Repo / AUR
    if shutil.which("pacman"):
        try:
            res_repo = subprocess.run(["pacman", "-Qq"], capture_output=True, text=True, check=False)
            total_pacman = len([p for p in res_repo.stdout.split("\n") if p.strip()])
            res_aur = subprocess.run(["pacman", "-Qm"], capture_output=True, text=True, check=False)
            aur_count = len([p for p in res_aur.stdout.split("\n") if p.strip()])
            stats.package_counts["repo"] = max(0, total_pacman - aur_count)
            stats.package_counts["aur"] = aur_count
        except Exception:
            pass

    # Flatpak
    if shutil.which("flatpak"):
        try:
            res_fp = subprocess.run(["flatpak", "list", "--app"], capture_output=True, text=True, check=False)
            fp_count = len([p for p in res_fp.stdout.split("\n") if p.strip()])
            stats.package_counts["flatpak"] = fp_count
        except Exception:
            pass

    # Snap
    if shutil.which("snap"):
        try:
            res_snap = subprocess.run(["snap", "list"], capture_output=True, text=True, check=False)
            lines = [l for l in res_snap.stdout.split("\n") if l.strip()]
            stats.package_counts["snap"] = max(0, len(lines) - 1)
        except Exception:
            pass

    # 3. Pending Updates Count
    if shutil.which("checkupdates"):
        try:
            res_up = subprocess.run(["checkupdates"], capture_output=True, text=True, check=False)
            stats.pending_updates = len([l for l in res_up.stdout.split("\n") if l.strip()])
        except Exception:
            pass
    elif shutil.which("paru"):
        try:
            res_up = subprocess.run(["paru", "-Qu"], capture_output=True, text=True, check=False)
            stats.pending_updates = len([l for l in res_up.stdout.split("\n") if l.strip()])
        except Exception:
            pass

    if stats.pending_updates > 0:
        stats.health_alerts.append(HealthCheckAlert(
            category="Updates",
            severity="info",
            message=f"{stats.pending_updates} package update(s) available.",
            suggestion="Run 'arjax update' to upgrade installed packages."
        ))

    # 4. Orphans Count & Alert
    detector = OrphanDetector()
    orphans = detector.find_orphans()
    stats.orphans_count = len(orphans)
    if stats.orphans_count > 0:
        stats.health_alerts.append(HealthCheckAlert(
            category="Dependencies",
            severity="warning" if stats.orphans_count > 10 else "info",
            message=f"Found {stats.orphans_count} orphaned package(s).",
            suggestion="Run 'arjax cleanup orphans' to clean up unreferenced dependencies."
        ))

    # 5. Failed Systemd Units Alert
    if shutil.which("systemctl"):
        try:
            res_sec = subprocess.run(
                ["systemctl", "--failed", "--plain", "--no-legend"],
                capture_output=True, text=True, check=False
            )
            failed_units = [u for u in res_sec.stdout.split("\n") if u.strip()]
            stats.failed_services_count = len(failed_units)
            if stats.failed_services_count > 0:
                stats.health_alerts.append(HealthCheckAlert(
                    category="Services",
                    severity="warning",
                    message=f"{stats.failed_services_count} systemd service(s) in failed state.",
                    suggestion="Check system logs with 'journalctl -xe' or 'systemctl status <unit>'."
                ))
        except Exception:
            pass

    return stats
