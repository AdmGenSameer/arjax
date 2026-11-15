"""Unit tests for ML-based smart package recommendations (TF-IDF)."""

from typer.testing import CliRunner

from arjax.intelligence.ml_suggestions import (
    MLPackageSuggester,
    RecommendationResult,
    SKLEARN_AVAILABLE,
)
from arjax.interfaces.cli import app


def test_recommendation_result_dataclass():
    """Verify RecommendationResult properties and percentage computation."""
    res = RecommendationResult(
        name="gimp",
        description="GNU Image Manipulation Program",
        provider="repo",
        similarity_score=0.954,
    )
    assert res.name == "gimp"
    assert res.match_percentage == 95

    # Check boundaries
    res_high = RecommendationResult("test", "desc", "prov", 1.5)
    assert res_high.match_percentage == 100

    res_low = RecommendationResult("test", "desc", "prov", 0.001)
    assert res_low.match_percentage == 1


def test_ml_suggester_initialization():
    """Verify suggester harvests corpus and initializes engines properly."""
    suggester = MLPackageSuggester()
    assert len(suggester.packages) > 0
    names = [p["name"].lower() for p in suggester.packages]
    assert "gimp" in names or "krita" in names


def test_sklearn_recommendations():
    """Verify scikit-learn recommendation for popular alternatives."""
    if not SKLEARN_AVAILABLE:
        return
    suggester = MLPackageSuggester(force_pure_python=False)
    assert suggester.use_sklearn is True

    recs = suggester.recommend("photoshop", top_n=3)
    assert len(recs) > 0
    rec_names = [r.name.lower() for r in recs]
    assert any(n in rec_names for n in ["gimp", "krita", "inkscape"])
    assert all(1 <= r.match_percentage <= 100 for r in recs)


def test_pure_python_fallback():
    """Verify pure-python engine works when scikit-learn is not used."""
    suggester = MLPackageSuggester(force_pure_python=True)
    assert suggester.use_sklearn is False

    recs = suggester.recommend("photoshop", top_n=3)
    assert len(recs) > 0
    rec_names = [r.name.lower() for r in recs]
    assert any(n in rec_names for n in ["gimp", "krita", "inkscape"])
    assert all(1 <= r.match_percentage <= 100 for r in recs)


def test_whatsapp_alternative_recommendation():
    """Verify alternative recommendation for proprietary chat apps."""
    suggester = MLPackageSuggester(force_pure_python=True)
    recs = suggester.recommend("whatsapp", top_n=3)
    assert len(recs) > 0
    rec_names = [r.name.lower() for r in recs]
    assert any("zapfast" in n or "whatsapp" in n for n in rec_names)


def test_empty_or_whitespace_query():
    """Verify empty or invalid queries return empty lists safely."""
    suggester = MLPackageSuggester()
    assert suggester.recommend("") == []
    assert suggester.recommend("   ") == []


def test_cli_search_smart_flag():
    """Verify arjax search --smart executes and displays recommendations."""
    runner = CliRunner()
    result = runner.invoke(app, ["search", "photoshop", "--smart"])
    assert result.exit_code == 0
    assert "ML Package Suggestions" in result.stdout or "gimp" in result.stdout


def test_cli_suggest_smart_flag():
    """Verify arjax suggest --smart outputs TF-IDF recommendations."""
    runner = CliRunner()
    result = runner.invoke(app, ["suggest", "photoshop", "--smart"])
    assert result.exit_code == 0
    assert "gimp" in result.stdout or "krita" in result.stdout


def test_tui_update_results_ml_fallback(monkeypatch):
    """Verify TUI _update_results surfaces ML alternatives on zero matches."""
    from unittest.mock import MagicMock
    from arjax.interfaces.tui import ArjaxTUIApp

    app = ArjaxTUIApp(engine=MagicMock())
    monkeypatch.setattr(app, "fetch_deps", MagicMock())

    mock_pkg_list = MagicMock()
    mock_details = MagicMock()
    mock_tree = MagicMock()

    def mock_query(selector, widget_type=None):
        if "pkg_list" in selector:
            return mock_pkg_list
        if "pkg_details" in selector:
            return mock_details
        if "dep_tree" in selector:
            return mock_tree
        return MagicMock()

    app.query_one = mock_query
    app._update_results("photoshop", [])

    assert "smart Linux alternative" in app.status_text
    assert mock_pkg_list.append.called


def test_tui_update_dep_tree_recommendations():
    """Verify smart alternatives node is added to dependency tree."""
    from unittest.mock import MagicMock
    from arjax.interfaces.tui import ArjaxTUIApp

    app = ArjaxTUIApp(engine=MagicMock())
    app.selected_pkg = {"name": "photoshop", "version": "1.0"}

    mock_tree = MagicMock()
    mock_rec_node = MagicMock()
    mock_tree.root.add.return_value = mock_rec_node

    def mock_query(selector, widget_type=None):
        if "dep_tree" in selector:
            return mock_tree
        return MagicMock()

    app.query_one = mock_query

    info = {
        "name": "photoshop",
        "dependencies": ["glibc"],
        "recommendations": [
            RecommendationResult("gimp", "Image editor", "repo", 0.96)
        ],
    }
    app._update_dep_tree("photoshop", info)

    assert any(
        "Smart Alternatives" in str(call)
        for call in mock_tree.root.add.call_args_list
    )
    assert mock_rec_node.add.called
