import pytest
from unittest.mock import MagicMock
from arjax.provider_engine import GoProviderEngine
from arjax.interfaces.tui import ArjaxTUIApp


def test_provider_engine_info_dependencies(monkeypatch):
    engine = GoProviderEngine()
    
    mock_response = {
        "name": "firefox",
        "version": "120.0-1",
        "description": "Fast Web Browser",
        "dependencies": ["glibc", "gcc-libs", "libpng"],
        "provider": "repository"
    }
    
    monkeypatch.setattr(engine, "_run", lambda cmd: mock_response)
    info = engine.info("firefox")
    assert info["name"] == "firefox"
    assert info["dependencies"] == ["glibc", "gcc-libs", "libpng"]


@pytest.mark.asyncio
async def test_tui_dep_tree_update():
    app = ArjaxTUIApp(engine=MagicMock())
    async with app.run_test() as pilot:
        app.selected_pkg = {"name": "testpkg", "version": "1.0.0"}
        info = {
            "name": "testpkg",
            "dependencies": ["dep-a", "dep-b"]
        }
        app._update_dep_tree("testpkg", info)
        tree = app.query_one("#dep_tree")
        children = [str(n.label) for n in tree.root.children]
        assert any("dep-a" in c for c in children)
        assert any("dep-b" in c for c in children)
