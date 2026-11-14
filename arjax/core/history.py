"""Transaction History & Rollback Management for Arjax."""

from dataclasses import dataclass, asdict
from datetime import datetime
import json
import os
import uuid
from typing import Any, Dict, List, Optional


@dataclass
class Transaction:
    id: str
    timestamp: str
    action: str  # "install", "remove", "rollback"
    package: str
    provider: str
    status: str  # "success", "failed"
    duration_ms: int
    error: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Transaction":
        return cls(
            id=data.get("id", ""),
            timestamp=data.get("timestamp", ""),
            action=data.get("action", "unknown"),
            package=data.get("package", ""),
            provider=data.get("provider", "repo"),
            status=data.get("status", "failed"),
            duration_ms=data.get("duration_ms", 0),
            error=data.get("error", ""),
        )


class TransactionManager:
    """Manages persistent transaction history and rollback operations."""

    def __init__(self, storage_path: Optional[str] = None):
        if storage_path:
            self.storage_path = storage_path
        else:
            base_dir = os.path.expanduser("~/.local/share/arjax")
            os.makedirs(base_dir, exist_ok=True)
            self.storage_path = os.path.join(base_dir, "history.json")

    def _load_history(self) -> List[Transaction]:
        if not os.path.exists(self.storage_path):
            return []
        try:
            with open(self.storage_path, "r", encoding="utf-8") as f:
                data = json.load(f)
                return [Transaction.from_dict(item) for item in data]
        except Exception:
            return []

    def _save_history(self, history: List[Transaction]) -> None:
        folder = os.path.dirname(self.storage_path)
        if folder:
            os.makedirs(folder, exist_ok=True)
        with open(self.storage_path, "w", encoding="utf-8") as f:
            json.dump([tx.to_dict() for tx in history], f, indent=2)

    def log_transaction(
        self,
        action: str,
        package: str,
        provider: str = "repo",
        success: bool = True,
        duration_ms: int = 0,
        error: str = "",
    ) -> Transaction:
        tx = Transaction(
            id=uuid.uuid4().hex[:8],
            timestamp=datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            action=action,
            package=package,
            provider=provider,
            status="success" if success else "failed",
            duration_ms=duration_ms,
            error=error,
        )
        history = self._load_history()
        history.insert(0, tx)  # Newest first
        self._save_history(history)
        return tx

    def get_history(self, limit: int = 50) -> List[Transaction]:
        history = self._load_history()
        return history[:limit]

    def get_transaction(self, tx_id: str) -> Optional[Transaction]:
        history = self._load_history()
        for tx in history:
            if tx.id == tx_id or tx.id.startswith(tx_id):
                return tx
        return None

    def rollback(self, tx_id: str, engine: Any) -> Dict[str, Any]:
        tx = self.get_transaction(tx_id)
        if not tx:
            return {"success": False, "error": f"Transaction '{tx_id}' not found"}

        if tx.status != "success":
            return {"success": False, "error": f"Cannot rollback failed transaction '{tx.id}'"}

        start_time = datetime.now()
        if tx.action == "install":
            res = engine.remove(tx.package)
            reverse_action = "rollback_remove"
        elif tx.action == "remove":
            res = engine.install(tx.package)
            reverse_action = "rollback_install"
        else:
            return {"success": False, "error": f"Unsupported rollback action '{tx.action}'"}

        duration = int((datetime.now() - start_time).total_seconds() * 1000)
        succ = res.get("success", False) if isinstance(res, dict) else bool(res)
        err = res.get("error", "") if isinstance(res, dict) else ""

        self.log_transaction(
            action=reverse_action,
            package=tx.package,
            provider=tx.provider,
            success=succ,
            duration_ms=duration,
            error=err,
        )

        return {
            "success": succ,
            "transaction_id": tx.id,
            "target_package": tx.package,
            "reversed_action": reverse_action,
            "error": err,
        }

    def clear_history(self) -> None:
        self._save_history([])
