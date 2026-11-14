import os
from pathlib import Path
import tempfile
import pytest
from arjax.system.completions import generate_completion_script, install_completion


def test_generate_completion_script():
    bash_script = generate_completion_script("bash")
    assert "_arjax_completion" in bash_script
    assert "complete -F _arjax_completion arjax" in bash_script

    zsh_script = generate_completion_script("zsh")
    assert "#compdef arjax" in zsh_script

    fish_script = generate_completion_script("fish")
    assert "complete -c arjax" in fish_script

    with pytest.raises(ValueError):
        generate_completion_script("invalid_shell")


def test_install_completion(monkeypatch):
    with tempfile.TemporaryDirectory() as tmpdir:
        monkeypatch.setattr("pathlib.Path.home", lambda: Path(tmpdir))
        path = install_completion("fish")
        assert os.path.exists(path)
        assert path.endswith("arjax.fish")
