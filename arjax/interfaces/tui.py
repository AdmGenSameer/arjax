"""Terminal User Interface (TUI) for Arjax."""

import shutil
import subprocess
import time
from pathlib import Path
from typing import Any, Dict, List, Optional

from textual import work
from textual.app import App, ComposeResult
from textual.containers import Container, Horizontal, Vertical
from textual.reactive import reactive
from textual.widgets import (
    Button,
    Footer,
    Header,
    Input,
    Label,
    ListItem,
    ListView,
    Static,
    Tree,
)

from arjax.core.history import TransactionManager
from arjax.installation.orchestrator import InstallationOrchestrator
from arjax.installation.providers import detect_distro_family
from arjax.installation.recipes import RecipeStore
from arjax.package_management.installed import installed_apps_manager
from arjax.provider_engine import GoProviderEngine


class PackageItem(ListItem):
    """List item representing a package."""

    def __init__(self, pkg_dict: dict):
        super().__init__()
        self.pkg = pkg_dict

    def compose(self) -> ComposeResult:
        name = self.pkg.get("name", "Unknown")
        prov = self.pkg.get("provider", "repo")
        ver = self.pkg.get("version", "")
        yield Label(
            f"📦 [bold cyan]{name}[/] [dim magenta]({prov})[/] [green]{ver}[/]"
        )


class ArjaxTUIApp(App):
    """Arjax Modern TUI Application."""

    CSS = """
    Screen {
        background: #1a1b26;
    }
    Header {
        background: #7aa2f7;
        color: #1a1b26;
        dock: top;
    }
    Footer {
        background: #24283b;
        color: #c0caf5;
        dock: bottom;
    }
    #search_input {
        dock: top;
        margin: 1 2;
        border: tall #7aa2f7;
        background: #24283b;
    }
    #main_container {
        layout: horizontal;
        height: 1fr;
        margin: 0 1;
    }
    #list_pane {
        width: 50%;
        border: round #7aa2f7;
        background: #1f2335;
        padding: 1;
    }
    #details_pane {
        width: 50%;
        border: round #bb9af7;
        background: #1f2335;
        padding: 1;
    }
    #pkg_details {
        height: auto;
        max-height: 8;
        margin-bottom: 1;
    }
    #dep_tree {
        height: 1fr;
        border: round #414868;
        background: #1a1b26;
    }
    #status_bar {
        height: 3;
        background: #24283b;
        color: #7dcfff;
        content-align: center middle;
        margin: 0 2 1 2;
        border: round #414868;
    }
    .action_btn {
        margin: 1 1;
    }
    """

    BINDINGS = [
        ("q", "quit", "Quit"),
        ("ctrl+c", "quit", "Quit"),
        ("i", "install_selected", "Install Package"),
        ("r", "remove_selected", "Remove Package"),
        ("/", "focus_search", "Focus Search"),
        ("d", "focus_deps", "Focus Dependency Tree"),
        ("h", "view_history", "Transaction History"),
        ("o", "scan_orphans", "Scan Orphans"),
        ("s", "view_stats", "System Dashboard"),
        ("u", "sync_recipes", "Sync Recipes"),
    ]

    selected_pkg = reactive(None)
    status_text = reactive(
        "Ready. Press '/' to search or type a package name (e.g. zapfast)."
    )

    def __init__(self, engine: Optional[GoProviderEngine] = None):
        super().__init__()
        self.engine = engine or GoProviderEngine()
        self._search_timer = None

    def watch_status_text(self, new_val: str) -> None:
        """Dynamically update status bar when status_text changes."""
        try:
            self.query_one("#status_bar", Static).update(new_val)
        except Exception:
            pass

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Input(
            placeholder=(
                "Search packages (e.g. zapfast, vlc) — "
                "Press Enter or pause to search..."
            ),
            id="search_input",
        )
        with Container(id="main_container"):
            with Vertical(id="list_pane"):
                yield Label("[bold cyan]SEARCH RESULTS[/]")
                yield ListView(id="pkg_list")
            with Vertical(id="details_pane"):
                yield Label(
                    "[bold magenta]PACKAGE DETAILS & DEPENDENCY TREE[/]"
                )
                yield Static(
                    "Select a package to view details.", id="pkg_details"
                )
                yield Tree("Dependencies", id="dep_tree")
                with Horizontal():
                    yield Button(
                        "Install (i)",
                        id="btn_install",
                        variant="success",
                        classes="action_btn",
                    )
                    yield Button(
                        "Remove (r)",
                        id="btn_remove",
                        variant="error",
                        classes="action_btn",
                    )
        yield Static(self.status_text, id="status_bar")
        yield Footer()

    def on_input_changed(self, event: Input.Changed) -> None:
        """Provide real-time typing feedback and debounce auto-search."""
        query = event.value.strip()
        if self._search_timer is not None:
            self._search_timer.stop()
            self._search_timer = None

        if not query:
            self.status_text = (
                "Ready. Press '/' to search or type a package name "
                "(e.g. zapfast, vlc)."
            )
            return

        self.status_text = (
            f"⌨ Typing: '{query}' — Press [bold]Enter[/bold] to search now, "
            "or pause for auto-search..."
        )
        self._search_timer = self.set_timer(
            0.5, lambda: self._trigger_search(query)
        )

    def on_input_submitted(self, event: Input.Submitted) -> None:
        """Search immediately when Enter is pressed."""
        query = event.value.strip()
        if self._search_timer is not None:
            self._search_timer.stop()
            self._search_timer = None
        if query:
            self._trigger_search(query)

    def _trigger_search(self, query: str) -> None:
        """Show active searching status and launch package search."""
        self.status_text = (
            f"⏳ Searching for '{query}' across recipes and packages..."
        )
        try:
            pkg_list = self.query_one("#pkg_list", ListView)
            pkg_list.clear()
            pkg_list.append(
                ListItem(
                    Label(
                        f"⏳ [bold cyan]Searching for '{query}'...[/bold cyan]"
                    )
                )
            )
        except Exception:
            pass
        self.query_packages(query)

    def _search_all_sources(self, query: str) -> List[Dict[str, Any]]:
        """Search across declarative recipes, Go engine, and system repos."""
        pkgs: List[Dict[str, Any]] = []
        seen = set()

        # 1. Search declarative recipes (e.g. zapfast, brave, vscode, docker)
        try:
            recipe_store = RecipeStore()
            q_lower = query.lower()
            for loc in recipe_store.load_all():
                recipe = loc.recipe
                aliases = [a.lower() for a in getattr(recipe, "aliases", [])]
                d_name = getattr(recipe, "display_name", None)
                d_match = d_name and q_lower in d_name.lower()
                name_match = (
                    q_lower in recipe.name.lower()
                    or d_match
                    or any(q_lower in a for a in aliases)
                )
                if name_match:
                    provs = (
                        list(recipe.providers.keys())
                        if getattr(recipe, "providers", None)
                        else ["recipe"]
                    )
                    prov_str = f"recipe ({', '.join(provs)})"
                    desc = (
                        recipe.description
                        or f"Declarative recipe for {recipe.name}"
                    )
                    p_info = {
                        "name": recipe.name,
                        "display_name": d_name or recipe.name,
                        "provider": prov_str,
                        "version": "latest",
                        "description": desc,
                        "repo": "arjax-recipes",
                        "is_recipe": True,
                    }
                    if recipe.name not in seen:
                        seen.add(recipe.name)
                        pkgs.append(p_info)
        except Exception:
            pass

        # 2. Search Go Provider Engine if available
        if self.engine.is_available():
            try:
                res = self.engine.search(query)
                for p in res.get("packages", []):
                    key = (p.get("name"), p.get("provider"))
                    if key not in seen and p.get("name") not in seen:
                        seen.add(key)
                        pkgs.append(p)
            except Exception:
                pass

        # 3. Fallback to native system package managers and Flatpak
        try:
            family = detect_distro_family()

            if family == "debian":
                from arjax.search.apt import search_apt
                try:
                    for item in search_apt(query)[:10]:
                        name, desc = item[0], item[1]
                        src = item[2] if len(item) > 2 else "apt"
                        if name not in seen:
                            seen.add(name)
                            pkgs.append({
                                "name": name,
                                "provider": src,
                                "version": "",
                                "description": desc,
                                "repo": src,
                            })
                except Exception:
                    pass
            elif family == "arch":
                from arjax.search.pacman import search_pacman
                from arjax.search.aur import search_aur
                try:
                    for item in search_pacman(query)[:10]:
                        name, desc = item[0], item[1]
                        src = item[2] if len(item) > 2 else "pacman"
                        if name not in seen:
                            seen.add(name)
                            pkgs.append({
                                "name": name,
                                "provider": src,
                                "version": "",
                                "description": desc,
                                "repo": src,
                            })
                except Exception:
                    pass
                try:
                    for item in search_aur(query)[:10]:
                        name, desc = item[0], item[1]
                        src = item[2] if len(item) > 2 else "aur"
                        if name not in seen:
                            seen.add(name)
                            pkgs.append({
                                "name": name,
                                "provider": src,
                                "version": "",
                                "description": desc,
                                "repo": src,
                            })
                except Exception:
                    pass
            elif family == "fedora":
                from arjax.search.dnf import search_dnf
                try:
                    for item in search_dnf(query)[:10]:
                        name, desc = item[0], item[1]
                        src = item[2] if len(item) > 2 else "dnf"
                        if name not in seen:
                            seen.add(name)
                            pkgs.append({
                                "name": name,
                                "provider": src,
                                "version": "",
                                "description": desc,
                                "repo": src,
                            })
                except Exception:
                    pass
            elif family == "suse":
                from arjax.search.zypper import search_zypper
                try:
                    for item in search_zypper(query)[:10]:
                        name, desc = item[0], item[1]
                        src = item[2] if len(item) > 2 else "zypper"
                        if name not in seen:
                            seen.add(name)
                            pkgs.append({
                                "name": name,
                                "provider": src,
                                "version": "",
                                "description": desc,
                                "repo": src,
                            })
                except Exception:
                    pass

            if shutil.which("flatpak"):
                from arjax.search.flatpak import search_flatpak
                try:
                    for item in search_flatpak(query)[:5]:
                        name, desc = item[0], item[1]
                        key = (name, "flatpak")
                        if key not in seen:
                            seen.add(key)
                            pkgs.append({
                                "name": name,
                                "provider": "flatpak",
                                "version": "",
                                "description": desc,
                                "repo": "flathub",
                            })
                except Exception:
                    pass
        except Exception:
            pass

        return pkgs

    @work(exclusive=True, thread=True)
    def query_packages(self, query: str) -> None:
        """Search across declarative recipes, Go engine, and system repos."""
        pkgs = self._search_all_sources(query)
        self.app.call_from_thread(self._update_results, query, pkgs)

    def _update_results(self, query: str, pkgs: list) -> None:
        """Update package list and auto-select first result."""
        pkg_list = self.query_one("#pkg_list", ListView)
        pkg_list.clear()
        if pkgs:
            for p in pkgs:
                pkg_list.append(PackageItem(p))
            self.status_text = (
                f"✔ Found {len(pkgs)} package(s) for '{query}'. "
                "Press [bold]i[/bold] to Install, [bold]r[/bold] to Remove."
            )
            self.selected_pkg = pkgs[0]
            self._render_details(pkgs[0])
        else:
            msg = (
                f"✖ [bold yellow]No packages found matching '{query}'.[/]"
            )
            pkg_list.append(ListItem(Label(msg)))
            self.status_text = (
                f"✖ No packages found for '{query}'. Try another search query."
            )
            self.selected_pkg = None
            try:
                self.query_one("#pkg_details", Static).update(
                    "No package selected."
                )
                tree = self.query_one("#dep_tree", Tree)
                tree.clear()
                tree.root.set_label("Dependencies")
            except Exception:
                pass

    def on_list_view_selected(self, event: ListView.Selected) -> None:
        if isinstance(event.item, PackageItem):
            self.selected_pkg = event.item.pkg
            self._render_details(event.item.pkg)

    def on_list_view_highlighted(self, event: ListView.Highlighted) -> None:
        if event.item and isinstance(event.item, PackageItem):
            self.selected_pkg = event.item.pkg
            self._render_details(event.item.pkg)

    def _render_details(self, pkg: dict) -> None:
        details = self.query_one("#pkg_details", Static)
        tree = self.query_one("#dep_tree", Tree)
        tree.clear()
        name = pkg.get("name", "")
        prov = pkg.get("provider", "")
        ver = pkg.get("version", "latest")
        desc = pkg.get("description", "No description available.")
        repo = pkg.get("repo", "N/A")

        text = (
            f"[bold cyan]Name:[/]        {name}\n"
            f"[bold magenta]Provider:[/]    {prov}\n"
            f"[bold green]Version:[/]     {ver}\n"
            f"[bold yellow]Repository:[/]  {repo}\n"
            f"[bold white]Description:[/] {desc}\n"
        )
        details.update(text)
        tree.root.set_label(f"📦 [bold cyan]{name}[/] [green]({ver})[/]")
        tree.root.expand()

        self.fetch_deps(name)

    @work(exclusive=True, thread=True)
    def fetch_deps(self, pkg_name: str) -> None:
        """Fetch and populate dependency/provider details for selected pkg."""
        strategies = []
        try:
            recipe = RecipeStore().find(pkg_name)
            if recipe and getattr(recipe, "providers", None):
                for prov_name, prov_conf in recipe.providers.items():
                    info_str = prov_name
                    if prov_conf.package:
                        info_str += f": {prov_conf.package}"
                    elif prov_conf.github_repo:
                        info_str += f": {prov_conf.github_repo}"
                    elif prov_conf.appimage_url:
                        info_str += (
                            f": {prov_conf.appimage_url.split('/')[-1]}"
                        )
                    strategies.append(info_str)
        except Exception:
            pass

        info = {"name": pkg_name}
        if strategies:
            info["dependencies"] = [
                f"Provider Strategy: {s}" for s in strategies
            ]
        elif self.engine.is_available():
            info = self.engine.info(pkg_name)
        else:
            info["dependencies"] = []

        self.app.call_from_thread(self._update_dep_tree, pkg_name, info)

    def _update_dep_tree(self, pkg_name: str, info: dict) -> None:
        if not self.selected_pkg or self.selected_pkg.get("name") != pkg_name:
            return
        tree = self.query_one("#dep_tree", Tree)
        tree.clear()
        ver = self.selected_pkg.get("version", "latest")
        tree.root.set_label(f"📦 [bold cyan]{pkg_name}[/] [green]({ver})[/]")

        deps = info.get("dependencies", [])
        if deps:
            for dep in deps:
                tree.root.add(f"🔗 [bold yellow]{dep}[/]")
        else:
            tree.root.add(
                "[dim italic]No explicit dependencies listed[/dim italic]"
            )
        tree.root.expand()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button clicks for Install and Remove."""
        if event.button.id == "btn_install":
            self.action_install_selected()
        elif event.button.id == "btn_remove":
            self.action_remove_selected()

    def action_install_selected(self) -> None:
        """Trigger installation of the currently selected package."""
        if not self.selected_pkg:
            self.status_text = (
                "[bold yellow]⚠ Please select a package from the search "
                "results first.[/bold yellow]"
            )
            return

        name = self.selected_pkg.get("name")
        prov = self.selected_pkg.get("provider", "repo")
        self.status_text = (
            f"⏳ Installing [bold cyan]{name}[/bold cyan] via {prov}... "
            "Please wait."
        )
        self.run_install(name, self.selected_pkg)

    @work(exclusive=True, thread=True)
    def run_install(self, pkg_name: str, pkg_dict: dict) -> None:
        """Perform installation via Go engine or Python Orchestrator."""
        succ = False
        err = ""
        prov = pkg_dict.get("provider", "repo")
        t0 = time.time()

        if pkg_dict.get("is_recipe") or not self.engine.is_available():
            try:
                orchestrator = InstallationOrchestrator()
                res = orchestrator.install(pkg_name)
                succ = res.success
                err = res.error or res.message
                prov = res.provider_name
            except Exception as e:
                succ = False
                err = str(e)
        else:
            res = self.engine.install(pkg_name)
            succ = (
                res.get("success", False) if isinstance(res, dict) else False
            )
            err = (
                res.get("error", "Unknown error")
                if isinstance(res, dict)
                else str(res)
            )
            prov = (
                res.get("provider", prov) if isinstance(res, dict) else prov
            )

        duration_ms = int((time.time() - t0) * 1000)

        try:
            TransactionManager().log_transaction(
                action="install",
                package=pkg_name,
                provider=prov,
                success=succ,
                duration_ms=duration_ms,
                error=err if not succ else "",
            )
        except Exception:
            pass

        msg = (
            f"✔ [bold green]Successfully installed {pkg_name} via {prov}![/]"
            if succ
            else f"✖ [bold red]Failed to install {pkg_name}: {err}[/]"
        )
        self.app.call_from_thread(self._set_status, msg)

    def action_remove_selected(self) -> None:
        """Trigger removal of the currently selected package."""
        if not self.selected_pkg:
            self.status_text = (
                "[bold yellow]⚠ Please select a package from the search "
                "results first.[/bold yellow]"
            )
            return

        name = self.selected_pkg.get("name")
        self.status_text = (
            f"⏳ Removing [bold red]{name}[/bold red]... Please wait."
        )
        self.run_remove(name, self.selected_pkg)

    @work(exclusive=True, thread=True)
    def run_remove(self, pkg_name: str, pkg_dict: dict) -> None:
        """Perform package removal across Go engine, AppImage, and PMs."""
        succ = False
        err = ""
        prov = pkg_dict.get("provider", "repo")
        t0 = time.time()

        if self.engine.is_available():
            res = self.engine.remove(pkg_name)
            if isinstance(res, dict) and res.get("success"):
                succ = True
                prov = res.get("provider", prov)

        if not succ:
            target_bin = Path.home() / ".local" / "bin" / pkg_name
            app_dir = Path.home() / ".local" / "share" / "applications"
            target_desktop = app_dir / f"{pkg_name}.desktop"
            found_local = False
            if target_bin.exists():
                try:
                    target_bin.unlink()
                    found_local = True
                except Exception as e:
                    err = f"Failed to delete {target_bin}: {e}"
            if target_desktop.exists():
                try:
                    target_desktop.unlink()
                    found_local = True
                except Exception:
                    pass

            if not found_local and (
                "flatpak" in prov.lower() or shutil.which("flatpak")
            ):
                try:
                    r = subprocess.run(
                        ["flatpak", "uninstall", "-y", pkg_name],
                        capture_output=True,
                        text=True,
                        check=False,
                    )
                    if r.returncode == 0:
                        succ = True
                        prov = "flatpak"
                except Exception:
                    pass

            if not found_local and not succ and (
                "snap" in prov.lower() or shutil.which("snap")
            ):
                try:
                    r = subprocess.run(
                        ["sudo", "snap", "remove", pkg_name],
                        capture_output=True,
                        text=True,
                        check=False,
                    )
                    if r.returncode == 0:
                        succ = True
                        prov = "snap"
                except Exception:
                    pass

            if not found_local and not succ:
                cmd = None
                if shutil.which("paru"):
                    cmd = ["paru", "-Rns", "--noconfirm", pkg_name]
                elif shutil.which("pacman"):
                    cmd = ["sudo", "pacman", "-Rns", "--noconfirm", pkg_name]
                elif shutil.which("apt-get"):
                    cmd = ["sudo", "apt-get", "remove", "-y", pkg_name]
                elif shutil.which("dnf"):
                    cmd = ["sudo", "dnf", "remove", "-y", pkg_name]
                elif shutil.which("zypper"):
                    cmd = ["sudo", "zypper", "remove", "-y", pkg_name]

                if cmd:
                    try:
                        r = subprocess.run(
                            cmd, capture_output=True, text=True, check=False
                        )
                        if r.returncode == 0:
                            succ = True
                        else:
                            err = (
                                r.stderr
                                or r.stdout
                                or f"Removal returned {r.returncode}"
                            )
                    except Exception as e:
                        err = str(e)

            if found_local:
                succ = True
                prov = "appimage/local"

        try:
            installed_apps_manager.remove_package(pkg_name)
        except Exception:
            pass

        duration_ms = int((time.time() - t0) * 1000)

        try:
            TransactionManager().log_transaction(
                action="remove",
                package=pkg_name,
                provider=prov,
                success=succ,
                duration_ms=duration_ms,
                error=err if not succ else "",
            )
        except Exception:
            pass

        msg = (
            f"✔ [bold green]Successfully removed {pkg_name}![/bold green]"
            if succ
            else f"✖ [bold red]Failed to remove {pkg_name}: "
                 f"{err or 'Package not found or could not be removed'}[/]"
        )
        self.app.call_from_thread(self._set_status, msg)

    def _set_status(self, msg: str) -> None:
        self.status_text = msg

    def action_focus_search(self) -> None:
        self.query_one("#search_input", Input).focus()

    def action_focus_deps(self) -> None:
        self.query_one("#dep_tree", Tree).focus()

    def action_view_history(self) -> None:
        mgr = TransactionManager()
        txs = mgr.get_history(limit=20)
        tree = self.query_one("#dep_tree", Tree)
        tree.clear()
        tree.root.set_label("📜 [bold magenta]TRANSACTION HISTORY[/]")
        if txs:
            for tx in txs:
                st = (
                    f"[green]{tx.status}[/green]"
                    if tx.status == "success"
                    else f"[red]{tx.status}[/red]"
                )
                tree.root.add(
                    f"⏱ [{tx.id}] [bold yellow]{tx.action.upper()}[/] "
                    f"[bold cyan]{tx.package}[/] ({st}) "
                    f"[dim]{tx.duration_ms}ms[/dim]"
                )
        else:
            tree.root.add(
                "[dim italic]No transaction history recorded yet[/dim italic]"
            )
        tree.root.expand()
        self.status_text = f"Showing {len(txs)} transaction history records."

    def action_scan_orphans(self) -> None:
        from arjax.core.orphans import OrphanDetector
        detector = OrphanDetector()
        orphans = detector.find_orphans()
        tree = self.query_one("#dep_tree", Tree)
        tree.clear()
        tree.root.set_label("🧹 [bold yellow]ORPHAN PACKAGES (DAG Analysis)[/]")
        if orphans:
            for o in orphans:
                tree.root.add(
                    f"🍃 [bold cyan]{o.get('name')}[/] "
                    f"[dim]({o.get('reason')})[/dim]"
                )
        else:
            tree.root.add(
                "[green italic]No orphaned packages detected.[/green italic]"
            )
        tree.root.expand()
        self.status_text = (
            f"Found {len(orphans)} orphaned package(s) via DAG analysis."
        )

    def action_view_stats(self) -> None:
        from arjax.system.stats import gather_system_statistics
        st = gather_system_statistics()
        tree = self.query_one("#dep_tree", Tree)
        tree.clear()
        tree.root.set_label(
            "📊 [bold cyan]SYSTEM STATISTICS & HEALTH DASHBOARD[/]"
        )

        # Disk
        disk_node = tree.root.add("💾 [bold]Disk Usage[/bold]")
        if st.root_disk:
            disk_node.add(
                f"Root (/): {st.root_disk.formatted_used()} / "
                f"{st.root_disk.formatted_total()} "
                f"({st.root_disk.percent_used:.1f}%)"
            )
        c_mb = st.arjax_cache_bytes / (1024**2)
        p_mb = st.pkg_cache_bytes / (1024**2)
        disk_node.add(f"Arjax Cache: {c_mb:.1f} MB")
        disk_node.add(f"Package Cache: {p_mb:.1f} MB")
        disk_node.expand()

        # Packages
        pkg_node = tree.root.add("📦 [bold]Package Counts[/bold]")
        for prov, count in st.package_counts.items():
            pkg_node.add(f"{prov.upper()}: {count} packages")
        pkg_node.expand()

        # Maintenance Status
        status_node = tree.root.add("⚡ [bold]Maintenance Status[/bold]")
        status_node.add(f"Pending Updates: {st.pending_updates}")
        status_node.add(f"Orphans: {st.orphans_count}")
        status_node.add(f"Failed Services: {st.failed_services_count}")
        status_node.expand()

        # Health Alerts
        health_node = tree.root.add("🏥 [bold]Health Diagnostics[/bold]")
        if st.health_alerts:
            for alert in st.health_alerts:
                sev_color = (
                    "red" if alert.severity == "critical"
                    else ("yellow" if alert.severity == "warning" else "blue")
                )
                health_node.add(
                    f"[{sev_color}][{alert.severity.upper()}][/{sev_color}] "
                    f"[bold]{alert.category}:[/bold] {alert.message}"
                )
        else:
            health_node.add("[green]✔ All health checks passed[/green]")
        health_node.expand()

        tree.root.expand()
        self.status_text = "System Statistics & Health Dashboard loaded."

    def action_sync_recipes(self) -> None:
        from arjax.recipes.sync import RecipeSyncManager
        sync_mgr = RecipeSyncManager()
        res = sync_mgr.sync_recipes()
        tree = self.query_one("#dep_tree", Tree)
        tree.clear()
        tree.root.set_label("📖 [bold cyan]DECLARATIVE RECIPES AUTO-SYNC[/]")
        if res.get("success"):
            tree.root.add(
                "[green]✔ Auto-sync complete! "
                f"{res.get('updated_count')} recipe(s) updated.[/green]"
            )
            for rec in res.get("updated_recipes", []):
                tree.root.add(f"  └─ 📦 [bold yellow]{rec}[/bold yellow]")
        else:
            tree.root.add(
                f"[red]✖ Recipe sync failed: {res.get('failed_recipes')}[/red]"
            )
        tree.root.expand()
        self.status_text = (
            f"Synced {res.get('updated_count', 0)} recipes from central "
            "registry."
        )


def launch_tui():
    """Launch the interactive TUI."""
    go_tui = shutil.which("arjax-provider")
    if not go_tui:
        import os
        candidate = os.path.join(
            os.path.dirname(__file__),
            "..",
            "..",
            "arjax-provider-engine",
            "bin",
            "arjax-provider",
        )
        if os.path.isfile(candidate):
            go_tui = candidate

    if go_tui:
        try:
            subprocess.run([go_tui, "tui"], check=False)
            return
        except Exception:
            pass

    # Fallback to Textual TUI
    app = ArjaxTUIApp()
    app.run()
