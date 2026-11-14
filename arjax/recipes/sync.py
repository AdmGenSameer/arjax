"""Central registry sync manager for auto-updating declarative installation recipes."""

import json
import os
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple
from urllib.request import urlopen, Request
from urllib.error import URLError, HTTPError
from arjax.installation.recipes import RecipeStore
from arjax.config.logging import get_logger

logger = get_logger(__name__)

DEFAULT_REGISTRY_URL = "https://raw.githubusercontent.com/AdmGenSameer/archpkg-helper/main/data/recipes_manifest.json"


class RecipeSyncManager:
    """Manages recipe versioning, central registry manifest queries, and recipe auto-updates."""

    def __init__(self, registry_url: Optional[str] = None, target_dir: Optional[Path] = None):
        self.registry_url = registry_url or DEFAULT_REGISTRY_URL
        if target_dir:
            self.target_dir = target_dir
        else:
            self.target_dir = Path(os.environ.get("XDG_CONFIG_HOME", Path.home() / ".config")) / "arjax" / "recipes"
        self.target_dir.mkdir(parents=True, exist_ok=True)
        self.store = RecipeStore()

    def fetch_manifest(self) -> Dict[str, Any]:
        """Fetch central recipe manifest JSON from registry URL."""
        req = Request(self.registry_url, headers={"User-Agent": "arjax-recipe-sync"})
        try:
            with urlopen(req, timeout=10) as response:
                content = response.read().decode("utf-8")
                return json.loads(content)
        except (URLError, HTTPError, json.JSONDecodeError) as e:
            logger.warning(f"Failed to fetch upstream recipe manifest from {self.registry_url}: {e}")
            # Fallback to local manifest if available
            local_manifest = Path(__file__).resolve().parent.parent.parent / "data" / "recipes_manifest.json"
            if local_manifest.exists():
                try:
                    return json.loads(local_manifest.read_text(encoding="utf-8"))
                except Exception:
                    pass
            return {"recipes": {}}

    def list_status(self) -> Dict[str, Any]:
        """Return full status mapping of local vs remote recipes."""
        manifest = self.fetch_manifest()
        return manifest.get("recipes", {})

    def check_updates(self) -> List[Dict[str, Any]]:
        """Compare local recipe versions against upstream central registry."""
        manifest = self.fetch_manifest()
        remote_recipes = manifest.get("recipes", {})
        updates_available: List[Dict[str, Any]] = []

        local_recipes = {r.recipe.name.lower(): r.recipe for r in self.store.load_all()}

        for name, remote_info in remote_recipes.items():
            norm_name = name.lower()
            remote_ver = remote_info.get("version", "1.0.0")
            local_recipe = local_recipes.get(norm_name)

            if not local_recipe:
                updates_available.append({
                    "name": name,
                    "status": "new",
                    "local_version": "None",
                    "remote_version": remote_ver,
                    "url": remote_info.get("url", "")
                })
            elif remote_ver != local_recipe.version:
                updates_available.append({
                    "name": name,
                    "status": "update",
                    "local_version": local_recipe.version,
                    "remote_version": remote_ver,
                    "url": remote_info.get("url", "")
                })

        return updates_available

    def sync_recipes(self, force: bool = False) -> Dict[str, Any]:
        """Download updated and new recipes from central registry into user recipes folder."""
        manifest = self.fetch_manifest()
        remote_recipes = manifest.get("recipes", {})

        updated = []
        failed = []

        for name, remote_info in remote_recipes.items():
            url = remote_info.get("url")
            if not url:
                continue

            dest_file = self.target_dir / f"{name}.yaml"
            try:
                req = Request(url, headers={"User-Agent": "arjax-recipe-sync"})
                with urlopen(req, timeout=10) as resp:
                    yaml_content = resp.read().decode("utf-8")
                    dest_file.write_text(yaml_content, encoding="utf-8")
                    updated.append(name)
                    logger.info(f"Updated recipe '{name}' to {dest_file}")
            except Exception as e:
                logger.error(f"Failed to pull recipe '{name}' from {url}: {e}")
                failed.append(name)

        return {
            "success": len(failed) == 0,
            "updated_count": len(updated),
            "failed_count": len(failed),
            "updated_recipes": updated,
            "failed_recipes": failed,
            "target_dir": str(self.target_dir)
        }
