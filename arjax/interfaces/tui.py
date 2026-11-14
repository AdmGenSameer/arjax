"""Interactive Terminal User Interface (TUI) for Arjax using Textual & Go Provider Engine."""

import shutil
import subprocess
from textual.app import App, ComposeResult
from textual.containers import Container, Horizontal, Vertical
from textual.widgets import Header, Footer, Input, Static, ListView, ListItem, Label, Button, LoadingIndicator, Tree
from textual.reactive import reactive
from textual import work

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
        yield Label(f"📦 [bold cyan]{name}[/] [dim magenta]({prov})[/] [green]{ver}[/]")


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
    status_text = reactive("Ready. Press '/' to search or type a query.")

    def __init__(self, engine: GoProviderEngine = None):
        super().__init__()
        self.engine = engine or GoProviderEngine()

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Input(placeholder="Search packages across providers (e.g. firefox, git, vlc)...", id="search_input")
        with Container(id="main_container"):
            with Vertical(id="list_pane"):
                yield Label("[bold cyan]SEARCH RESULTS[/]")
                yield ListView(id="pkg_list")
            with Vertical(id="details_pane"):
                yield Label("[bold magenta]PACKAGE DETAILS & DEPENDENCY TREE[/]")
                yield Static("Select a package to view details.", id="pkg_details")
                yield Tree("Dependencies", id="dep_tree")
                with Horizontal():
                    yield Button("Install (i)", id="btn_install", variant="success", classes="action_btn")
                    yield Button("Remove (r)", id="btn_remove", variant="error", classes="action_btn")
        yield Static(self.status_text, id="status_bar")
        yield Footer()

    def on_input_submitted(self, event: Input.Submitted) -> None:
        query = event.value.strip()
        if query:
            self.status_text = f"Searching for '{query}' across providers..."
            self.query_packages(query)

    @work(exclusive=True, thread=True)
    def query_packages(self, query: str) -> None:
        res = self.engine.search(query)
        pkgs = res.get("packages", [])
        self.app.call_from_thread(self._update_results, query, pkgs)

    def _update_results(self, query: str, pkgs: list) -> None:
        pkg_list = self.query_one("#pkg_list", ListView)
        pkg_list.clear()
        for p in pkgs:
            pkg_list.append(PackageItem(p))
        self.status_text = f"Found {len(pkgs)} packages for '{query}'"
        if pkgs:
            self.selected_pkg = pkgs[0]
            self._render_details(pkgs[0])

    def on_list_view_selected(self, event: ListView.Selected) -> None:
        if isinstance(event.item, PackageItem):
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
        info = self.engine.info(pkg_name)
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
                tree.root.add(f"🔗 [bold yellow]{dep}[/] [dim](direct dependency)[/]")
        else:
            tree.root.add("[dim italic]No explicit dependencies listed[/dim italic]")
        tree.root.expand()

    def action_focus_search(self) -> None:
        self.query_one("#search_input", Input).focus()

    def action_focus_deps(self) -> None:
        self.query_one("#dep_tree", Tree).focus()

    def action_view_history(self) -> None:
        from arjax.core.history import TransactionManager
        mgr = TransactionManager()
        txs = mgr.get_history(limit=20)
        tree = self.query_one("#dep_tree", Tree)
        tree.clear()
        tree.root.set_label("📜 [bold magenta]TRANSACTION HISTORY[/]")
        if txs:
            for tx in txs:
                st = f"[green]{tx.status}[/green]" if tx.status == "success" else f"[red]{tx.status}[/red]"
                tree.root.add(f"⏱ [{tx.id}] [bold yellow]{tx.action.upper()}[/] [bold cyan]{tx.package}[/] ({st}) [dim]{tx.duration_ms}ms[/dim]")
        else:
            tree.root.add("[dim italic]No transaction history recorded yet[/dim italic]")
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
                tree.root.add(f"🍃 [bold cyan]{o.get('name')}[/] [dim]({o.get('reason')})[/dim]")
        else:
            tree.root.add("[green italic]No orphaned packages detected.[/green italic]")
        tree.root.expand()
        self.status_text = f"Found {len(orphans)} orphaned package(s) via DAG analysis."

    def action_view_stats(self) -> None:
        from arjax.system.stats import gather_system_statistics
        st = gather_system_statistics()
        tree = self.query_one("#dep_tree", Tree)
        tree.clear()
        tree.root.set_label("📊 [bold cyan]SYSTEM STATISTICS & HEALTH DASHBOARD[/]")

        # Disk
        disk_node = tree.root.add("💾 [bold]Disk Usage[/bold]")
        if st.root_disk:
            disk_node.add(f"Root (/): {st.root_disk.formatted_used()} / {st.root_disk.formatted_total()} ({st.root_disk.percent_used:.1f}%)")
        disk_node.add(f"Arjax Cache: {st.arjax_cache_bytes / (1024**2):.1f} MB")
        disk_node.add(f"Package Cache: {st.pkg_cache_bytes / (1024**2):.1f} MB")
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
                sev_color = "red" if alert.severity == "critical" else ("yellow" if alert.severity == "warning" else "blue")
                health_node.add(f"[{sev_color}][{alert.severity.upper()}][/{sev_color}] [bold]{alert.category}:[/bold] {alert.message}")
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
            tree.root.add(f"[green]✔ Auto-sync complete! {res.get('updated_count')} recipe(s) updated.[/green]")
            for rec in res.get("updated_recipes", []):
                tree.root.add(f"  └─ 📦 [bold yellow]{rec}[/bold yellow]")
        else:
            tree.root.add(f"[red]✖ Recipe sync failed: {res.get('failed_recipes')}[/red]")
        tree.root.expand()
        self.status_text = f"Synced {res.get('updated_count', 0)} recipes from central registry."

    def action_install_selected(self) -> None:
        if self.selected_pkg:
            name = self.selected_pkg.get("name")
            prov = self.selected_pkg.get("provider")
            self.status_text = f"Installing {name} via {prov}..."
            self.run_install(name)

    @work(exclusive=True, thread=True)
    def run_install(self, pkg_name: str) -> None:
        res = self.engine.install(pkg_name)
        succ = res.get("success", False)
        msg = f"Successfully installed {pkg_name}!" if succ else f"Failed to install {pkg_name}: {res.get('error', 'unknown')}"
        self.app.call_from_thread(self._set_status, msg)

    def _set_status(self, msg: str) -> None:
        self.status_text = msg
        self.query_one("#status_bar", Static).update(msg)


def launch_tui():
    """Launch the interactive TUI."""
    # Check if native Go TUI is compiled and available
    go_tui = shutil.which("arjax-provider")
    if not go_tui:
        import os
        candidate = os.path.join(os.path.dirname(__file__), "..", "..", "arjax-provider-engine", "bin", "arjax-provider")
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
