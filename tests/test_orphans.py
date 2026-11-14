import pytest
from unittest.mock import MagicMock
from arjax.core.orphans import OrphanDetector, PackageNode


def test_orphan_detection_dag_logic():
    detector = OrphanDetector()

    # Construct mock DAG graph
    nodes = {
        "app-a": PackageNode(name="app-a", is_explicit=True, dependencies=["lib-x"]),
        "app-b": PackageNode(name="app-b", is_explicit=True, dependencies=["lib-x"]),
        "lib-x": PackageNode(name="lib-x", is_explicit=False, required_by={"app-a", "app-b"}),
        "lib-orphan": PackageNode(name="lib-orphan", is_explicit=False, required_by=set())
    }

    detector.build_dag = lambda: nodes
    orphans = detector.find_orphans()

    orphan_names = [o["name"] for o in orphans]
    assert "lib-orphan" in orphan_names
    assert "lib-x" not in orphan_names
    assert "app-a" not in orphan_names


def test_orphan_cleanup_dry_run():
    detector = OrphanDetector()
    res = detector.clean_orphans(["lib-orphan"], dry_run=True)
    assert res["success"] is True
    assert res["dry_run"] is True
    assert res["count"] == 1
    assert "lib-orphan" in res["orphans_removed"]
