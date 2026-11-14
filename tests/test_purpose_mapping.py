"""Test to validate data/purpose_mapping.yaml integrity."""
from pathlib import Path
from scripts.validate_mapping import validate_mapping_file


def test_purpose_mapping_file():
    root = Path(__file__).resolve().parent.parent
    mapping_file = root / "data" / "purpose_mapping.yaml"
    assert mapping_file.exists(), f"Missing purpose mapping at {mapping_file}"
    assert validate_mapping_file(str(mapping_file)) is True

