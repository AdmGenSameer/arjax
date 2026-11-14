import json
import tempfile
from pathlib import Path
import pytest
from arjax.installation.models import Recipe
from arjax.installation.recipes import RecipeStore
from arjax.recipes.sync import RecipeSyncManager


def test_recipe_version_parsing(tmp_path):
    recipe_file = tmp_path / "testpkg.yaml"
    recipe_file.write_text("""
name: testpkg
version: "2.1.0"
description: Test recipe
providers:
  repository:
    package: testpkg
""", encoding="utf-8")

    store = RecipeStore(extra_dirs=[tmp_path])
    recipe = store.find("testpkg")
    assert recipe is not None
    assert recipe.name == "testpkg"
    assert recipe.version == "2.1.0"


def test_recipe_sync_check_updates(tmp_path, monkeypatch):
    mock_manifest = {
        "recipes": {
            "testpkg": {
                "name": "testpkg",
                "version": "2.5.0",
                "url": "https://example.com/testpkg.yaml"
            }
        }
    }

    sync_mgr = RecipeSyncManager(target_dir=tmp_path)
    monkeypatch.setattr(sync_mgr, "fetch_manifest", lambda: mock_manifest)

    updates = sync_mgr.check_updates()
    assert len(updates) == 1
    assert updates[0]["name"] == "testpkg"
    assert updates[0]["remote_version"] == "2.5.0"


def test_recipe_sync_pull(tmp_path, monkeypatch):
    mock_manifest = {
        "recipes": {
            "demo": {
                "name": "demo",
                "version": "1.0.0",
                "url": "https://example.com/demo.yaml"
            }
        }
    }

    fake_yaml = "name: demo\nversion: '1.0.0'\ndescription: Demo recipe\n"

    class MockResponse:
        def __enter__(self):
            return self
        def __exit__(self, *args):
            pass
        def read(self):
            return fake_yaml.encode("utf-8")

    monkeypatch.setattr("arjax.recipes.sync.urlopen", lambda req, timeout=10: MockResponse())

    sync_mgr = RecipeSyncManager(target_dir=tmp_path)
    monkeypatch.setattr(sync_mgr, "fetch_manifest", lambda: mock_manifest)

    res = sync_mgr.sync_recipes()
    assert res["success"] is True
    assert res["updated_count"] == 1
    assert (tmp_path / "demo.yaml").exists()
