import pytest
from arjax.system.stats import gather_system_statistics, SystemStatistics, HealthCheckAlert


def test_gather_system_statistics():
    stats = gather_system_statistics()
    assert isinstance(stats, SystemStatistics)
    assert isinstance(stats.health_alerts, list)
    if stats.root_disk:
        assert stats.root_disk.total_bytes > 0
        assert stats.root_disk.percent_used >= 0.0


def test_health_check_alert():
    alert = HealthCheckAlert(
        category="Storage",
        severity="warning",
        message="High disk usage",
        suggestion="Clean cache"
    )
    assert alert.category == "Storage"
    assert alert.severity == "warning"
