import tempfile
import pytest
from unittest.mock import MagicMock
from arjax.core.history import TransactionManager, Transaction


def test_transaction_logging():
    with tempfile.NamedTemporaryFile(suffix=".json") as tmp:
        mgr = TransactionManager(storage_path=tmp.name)
        tx = mgr.log_transaction(
            action="install",
            package="neovim",
            provider="repo",
            success=True,
            duration_ms=450
        )
        assert tx.package == "neovim"
        assert tx.action == "install"
        assert tx.status == "success"
        assert tx.duration_ms == 450

        history = mgr.get_history()
        assert len(history) == 1
        assert history[0].package == "neovim"

        found = mgr.get_transaction(tx.id)
        assert found is not None
        assert found.id == tx.id


def test_transaction_rollback():
    with tempfile.NamedTemporaryFile(suffix=".json") as tmp:
        mgr = TransactionManager(storage_path=tmp.name)
        tx = mgr.log_transaction(
            action="install",
            package="htop",
            provider="repo",
            success=True,
            duration_ms=200
        )

        mock_engine = MagicMock()
        mock_engine.remove.return_value = {"success": True, "package": "htop"}

        res = mgr.rollback(tx.id, mock_engine)
        assert res["success"] is True
        assert res["target_package"] == "htop"
        assert res["reversed_action"] == "rollback_remove"
        mock_engine.remove.assert_called_once_with("htop")
