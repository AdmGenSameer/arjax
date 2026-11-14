"""Tests ensuring TUI is the default interface in CLI."""
import sys
from unittest.mock import patch, MagicMock
import pytest
from arjax.interfaces.cli import main, search
from arjax.config.manager import UserConfig


def test_main_defaults_to_tui_when_no_args():
    """Verify `arjax` with no arguments launches TUI by default."""
    with patch.object(sys, "argv", ["arjax"]), \
         patch("sys.stdin.isatty", return_value=True), \
         patch("arjax.interfaces.tui.launch_tui") as mock_launch:
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        mock_launch.assert_called_once()


def test_main_supports_explicit_tui_flag():
    """Verify `arjax --tui` launches TUI."""
    with patch.object(sys, "argv", ["arjax", "--tui"]), \
         patch("arjax.interfaces.tui.launch_tui") as mock_launch:
        with pytest.raises(SystemExit) as exc_info:
            main()
        assert exc_info.value.code == 0
        mock_launch.assert_called_once()


def test_search_without_query_launches_tui():
    """Verify `arjax search` with no arguments opens TUI rather than failing."""
    with patch("arjax.interfaces.tui.launch_tui") as mock_launch:
        search(query=None)
        mock_launch.assert_called_once()


def test_search_with_tui_flag_launches_tui():
    """Verify `arjax search <pkg> --tui` opens TUI."""
    with patch("arjax.interfaces.tui.launch_tui") as mock_launch:
        search(query=["firefox"], tui=True)
        mock_launch.assert_called_once()


def test_default_config_interface_is_tui():
    """Verify default UserConfig has default_interface set to 'tui'."""
    cfg = UserConfig()
    assert cfg.default_interface == "tui"
