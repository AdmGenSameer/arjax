"""Parallel Package Installation Engine with Rich progress bars."""

from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Any, Dict, List, Optional
from rich.progress import Progress, SpinnerColumn, TextColumn, BarColumn, TimeElapsedColumn
from arjax.provider_engine import GoProviderEngine
from arjax.core.errors import suggest_alternative_providers
from arjax.config.logging import get_logger

logger = get_logger(__name__)


class ParallelInstaller:
    """Manages parallel multi-package installations with concurrent worker pool and Rich progress bars."""

    def __init__(self, engine: Optional[GoProviderEngine] = None, max_workers: int = 3):
        self.engine = engine or GoProviderEngine()
        self.max_workers = max_workers

    def install_packages(
        self,
        package_names: List[str],
        provider: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """Install multiple packages concurrently with progress bars for each package."""
        if not package_names:
            return []

        results: List[Dict[str, Any]] = []
        logger.info(f"Starting parallel installation for packages {package_names} (max_workers={self.max_workers})")

        with Progress(
            SpinnerColumn(),
            TextColumn("[bold blue]{task.description}"),
            BarColumn(),
            TimeElapsedColumn(),
        ) as progress:

            tasks: Dict[str, Any] = {}
            for name in package_names:
                task_id = progress.add_task(f"Waiting: {name}", total=100)
                tasks[name] = task_id

            def _install_worker(name: str) -> Dict[str, Any]:
                task_id = tasks[name]
                progress.update(task_id, description=f"Installing {name}...", completed=30)
                
                try:
                    res = self.engine.install(name)
                    succ = res.get("success", False) if isinstance(res, dict) else False
                    err = res.get("error", "") if isinstance(res, dict) else ""
                    prov = res.get("provider", provider or "repo") if isinstance(res, dict) else "repo"

                    if succ:
                        progress.update(task_id, completed=100, description=f"[bold green]✔ Installed {name}[/bold green]")
                        logger.info(f"Parallel install succeeded for {name}")
                        return {"package": name, "success": True, "provider": prov, "error": ""}
                    else:
                        progress.update(task_id, completed=100, description=f"[bold red]✖ Failed {name}[/bold red]")
                        logger.error(f"Parallel install failed for {name}: {err}")
                        suggestions = suggest_alternative_providers(name, prov, self.engine)
                        return {
                            "package": name,
                            "success": False,
                            "provider": prov,
                            "error": err or "Installation failed",
                            "suggestions": suggestions
                        }
                except Exception as e:
                    progress.update(task_id, completed=100, description=f"[bold red]✖ Failed {name}[/bold red]")
                    logger.error(f"Parallel install exception for {name}: {e}")
                    suggestions = suggest_alternative_providers(name, provider or "repo", self.engine)
                    return {
                        "package": name,
                        "success": False,
                        "provider": provider or "repo",
                        "error": str(e),
                        "suggestions": suggestions
                    }

            with ThreadPoolExecutor(max_workers=min(self.max_workers, len(package_names))) as executor:
                future_to_name = {executor.submit(_install_worker, name): name for name in package_names}
                for future in as_completed(future_to_name):
                    res = future.result()
                    results.append(res)

        return results
