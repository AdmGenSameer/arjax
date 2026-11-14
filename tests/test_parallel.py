import pytest
from unittest.mock import MagicMock
from arjax.installation.parallel import ParallelInstaller


def test_parallel_installer_success():
    mock_engine = MagicMock()
    mock_engine.install.side_effect = lambda pkg: {"success": True, "provider": "repo", "error": ""}
    mock_engine.search.return_value = {"packages": []}

    installer = ParallelInstaller(engine=mock_engine, max_workers=2)
    results = installer.install_packages(["pkg1", "pkg2", "pkg3"])

    assert len(results) == 3
    assert all(r["success"] is True for r in results)
    pkgs = [r["package"] for r in results]
    assert "pkg1" in pkgs
    assert "pkg2" in pkgs
    assert "pkg3" in pkgs


def test_parallel_installer_failure_suggestions():
    mock_engine = MagicMock()
    def mock_install(pkg):
        if pkg == "pkg1":
            return {"success": True, "provider": "repo", "error": ""}
        return {"success": False, "provider": "repo", "error": "Package missing"}
    
    mock_engine.install.side_effect = mock_install
    mock_engine.search.return_value = {
        "packages": [{"name": "pkg2", "provider": "flatpak"}]
    }

    installer = ParallelInstaller(engine=mock_engine, max_workers=2)
    results = installer.install_packages(["pkg1", "pkg2"])

    res_map = {r["package"]: r for r in results}
    assert res_map["pkg1"]["success"] is True
    assert res_map["pkg2"]["success"] is False
    assert len(res_map["pkg2"]["suggestions"]) > 0
    assert res_map["pkg2"]["suggestions"][0]["provider"] == "flatpak"
