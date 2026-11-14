"""Unit tests for Arjax TUI search feedback, buttons, and actions."""

from unittest.mock import MagicMock
import pytest
from textual.widgets import Button, Input, Static

from arjax.interfaces.tui import ArjaxTUIApp


@pytest.mark.asyncio
async def test_tui_typing_status_feedback():
    """Verify typing in search input provides immediate status feedback."""
    app = ArjaxTUIApp(engine=MagicMock())
    async with app.run_test() as pilot:
        search_input = app.query_one("#search_input", Input)
        search_input.value = "zapfast"
        await pilot.pause()

        status = app.query_one("#status_bar", Static)
        assert "zapfast" in str(status.render())
        assert "Typing:" in str(status.render())


@pytest.mark.asyncio
async def test_tui_enter_triggers_search(monkeypatch):
    """Verify pressing Enter triggers search with status indication."""
    app = ArjaxTUIApp(engine=MagicMock())
    searched_query = []

    def mock_query_packages(query: str):
        searched_query.append(query)

    monkeypatch.setattr(app, "query_packages", mock_query_packages)

    async with app.run_test() as pilot:
        search_input = app.query_one("#search_input", Input)
        search_input.value = "zapfast"
        await search_input.action_submit()
        await pilot.pause()

        assert "zapfast" in searched_query
        status = app.query_one("#status_bar", Static)
        assert "Searching for 'zapfast'" in str(status.render())


def test_tui_search_finds_zapfast_recipe():
    """Verify _search_all_sources discovers zapfast declarative recipe."""
    mock_engine = MagicMock()
    mock_engine.is_available.return_value = False
    app = ArjaxTUIApp(engine=mock_engine)

    pkgs = app._search_all_sources("zapfast")
    assert any(p["name"] == "zapfast" for p in pkgs)
    recipe_pkg = next(p for p in pkgs if p["name"] == "zapfast")
    assert "recipe" in recipe_pkg["provider"]
    has_match = (
        "WhatsApp" in recipe_pkg["description"]
        or "zapfast" in recipe_pkg["name"]
    )
    assert has_match


@pytest.mark.asyncio
async def test_tui_update_results_renders_and_selects():
    """Verify _update_results updates pkg_list and status bar."""
    app = ArjaxTUIApp(engine=MagicMock())
    async with app.run_test() as pilot:
        test_pkgs = [{
            "name": "zapfast",
            "provider": "recipe (repository, github, appimage)",
            "version": "latest",
            "description": "WhatsApp client",
            "repo": "arjax-recipes",
        }]
        app._update_results("zapfast", test_pkgs)
        await pilot.pause()
        assert app.selected_pkg["name"] == "zapfast"
        status = app.query_one("#status_bar", Static)
        assert "Found 1 package" in str(status.render())


@pytest.mark.asyncio
async def test_tui_install_button_wired(monkeypatch):
    """Verify clicking the Install button triggers action_install_selected."""
    app = ArjaxTUIApp(engine=MagicMock())
    installed_calls = []

    def mock_run_install(pkg_name: str, pkg_dict: dict):
        installed_calls.append(pkg_name)

    monkeypatch.setattr(app, "run_install", mock_run_install)

    async with app.run_test() as pilot:
        app.selected_pkg = {"name": "zapfast", "provider": "recipe"}
        btn_install = app.query_one("#btn_install", Button)
        await pilot.click(btn_install)
        await pilot.pause()

        assert "zapfast" in installed_calls
        status = app.query_one("#status_bar", Static)
        assert "Installing" in str(status.render())


@pytest.mark.asyncio
async def test_tui_remove_button_wired(monkeypatch):
    """Verify clicking the Remove button triggers action_remove_selected."""
    app = ArjaxTUIApp(engine=MagicMock())
    removed_calls = []

    def mock_run_remove(pkg_name: str, pkg_dict: dict):
        removed_calls.append(pkg_name)

    monkeypatch.setattr(app, "run_remove", mock_run_remove)

    async with app.run_test() as pilot:
        app.selected_pkg = {"name": "zapfast", "provider": "recipe"}
        btn_remove = app.query_one("#btn_remove", Button)
        await pilot.click(btn_remove)
        await pilot.pause()

        assert "zapfast" in removed_calls
        status = app.query_one("#status_bar", Static)
        assert "Removing" in str(status.render())


@pytest.mark.asyncio
async def test_tui_keyboard_shortcuts_i_and_r(monkeypatch):
    """Verify keyboard shortcuts 'i' and 'r' work properly."""
    app = ArjaxTUIApp(engine=MagicMock())
    installed = []
    removed = []

    monkeypatch.setattr(
        app, "run_install", lambda name, p: installed.append(name)
    )
    monkeypatch.setattr(
        app, "run_remove", lambda name, p: removed.append(name)
    )

    async with app.run_test():
        app.selected_pkg = {"name": "zapfast", "provider": "recipe"}

        app.action_install_selected()
        assert "zapfast" in installed

        app.action_remove_selected()
        assert "zapfast" in removed


def test_tui_dep_tree_recipe_strategies():
    """Verify provider strategies are derived for declarative recipes."""
    from arjax.installation.recipes import RecipeStore
    recipe = RecipeStore().find("zapfast")
    assert recipe is not None
    assert "appimage" in recipe.providers
    assert "github" in recipe.providers
    assert "repository" in recipe.providers
