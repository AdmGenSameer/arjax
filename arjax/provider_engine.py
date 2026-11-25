import json
import os
import shutil
import subprocess
from time import time
from typing import Any, Dict, List, Optional
from arjax.core.history import TransactionManager


class GoProviderEngine:
    """Python bridge to arjax-provider Go engine with fallback discovery."""

    def __init__(self, binary_path: Optional[str] = None, history_manager: Optional[TransactionManager] = None):
        self.binary_path = binary_path or self._discover_binary()
        self.history_manager = history_manager or TransactionManager()

    def _discover_binary(self) -> str:
        # Check PATH
        path_bin = shutil.which("arjax-provider")
        if path_bin:
            return path_bin

        # Check local build directory
        here = os.path.dirname(os.path.abspath(__file__))
        candidate = os.path.abspath(os.path.join(here, "..", "arjax-provider-engine", "bin", "arjax-provider"))
        if os.path.isfile(candidate) and os.access(candidate, os.X_OK):
            return candidate

        # Check user bin
        user_bin = os.path.expanduser("~/.local/bin/arjax-provider")
        if os.path.isfile(user_bin) and os.access(user_bin, os.X_OK):
            return user_bin

        return "arjax-provider"

    def is_available(self) -> bool:
        """Check if Go provider engine binary is executable."""
        if os.path.isabs(self.binary_path) and os.path.isfile(self.binary_path):
            return os.access(self.binary_path, os.X_OK)
        return shutil.which(self.binary_path) is not None

    def _run(self, cmd: List[str]) -> Dict[str, Any]:
        try:
            full_cmd = [self.binary_path] + cmd + ["--json"]
            res = subprocess.run(full_cmd, capture_output=True, text=True, check=False)
            if res.stdout:
                return json.loads(res.stdout)
            return {"error": res.stderr or "No output returned from provider engine"}
        except Exception as e:
            return {"error": str(e)}

    def search(self, query: str) -> Dict[str, Any]:
        return self._run(["search", query])

    def install(self, package_name: str) -> Dict[str, Any]:
        t0 = time()
        res = self._run(["install", package_name])
        duration_ms = int((time() - t0) * 1000)
        succ = res.get("success", False) if isinstance(res, dict) else False
        prov = res.get("provider", "repo") if isinstance(res, dict) else "repo"
        err = res.get("error", "") if isinstance(res, dict) else str(res)
        self.history_manager.log_transaction(
            action="install",
            package=package_name,
            provider=prov,
            success=succ,
            duration_ms=duration_ms,
            error=err
        )
        return res

    def remove(self, package_name: str) -> Dict[str, Any]:
        t0 = time()
        res = self._run(["remove", package_name])
        duration_ms = int((time() - t0) * 1000)
        succ = res.get("success", False) if isinstance(res, dict) else False
        prov = res.get("provider", "repo") if isinstance(res, dict) else "repo"
        err = res.get("error", "") if isinstance(res, dict) else str(res)
        self.history_manager.log_transaction(
            action="remove",
            package=package_name,
            provider=prov,
            success=succ,
            duration_ms=duration_ms,
            error=err
        )
        return res

    def info(self, package_name: str) -> Dict[str, Any]:
        return self._run(["info", package_name])

    def batch_install(self, package_names: List[str]) -> Dict[str, Any]:
        return self._run(["batch-install"] + package_names)

    def update(self) -> Dict[str, Any]:
        """Check for updates across all supported providers."""
        return self._run(["update"])

    def upgrade(self, package_names: Optional[List[str]] = None) -> Dict[str, Any]:
        """Apply updates safely across all supported providers."""
        args = ["upgrade"]
        if package_names:
            args.extend(package_names)
        return self._run(args)

