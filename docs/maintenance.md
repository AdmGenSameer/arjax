# 🧹 Maintenance, Health & Diagnostics

Arjax includes system maintenance tools to keep your package environment clean and reliable.

---

## 1. System Statistics & Diagnostics (`arjax stats`)

Inspect storage consumption, package counts, and system health checks:

```bash
arjax stats
```

Outputs:
* **Disk Usage**: Root filesystem free space, Arjax cache size, and native package cache.
* **Package Counts**: Total packages installed by format (APT, Pacman, Flatpak, Snap).
* **Maintenance Alerts**: Pending updates, orphan package count, and failed systemd services.
* **Diagnostic Warnings**: Storage warnings or broken mirror alerts.

---

## 2. DAG-Based Orphan Package Cleanup

Arjax constructs a Directed Acyclic Graph (DAG) of installed packages and detects orphaned dependencies that are no longer required:

```bash
# Preview what would be removed without deleting
arjax cleanup orphans --dry-run

# Safely purge orphaned packages and reclaim disk space
arjax cleanup orphans
```

---

## 3. Transaction History & Rollback

Every installation and removal operation is logged with a unique transaction ID and execution timestamp:

```bash
# View chronological history
arjax history list

# Rollback a specific transaction
arjax history rollback <tx_id>
```

---

## 4. Cache Management

Reclaim disk space by clearing expired search cache databases and package download caches:

```bash
arjax cleanup cache
```

