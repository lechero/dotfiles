"""Session history tracking for AgentX."""
from __future__ import annotations

import json
from dataclasses import asdict, dataclass
from datetime import datetime
from pathlib import Path
from typing import Any, Dict

from .config import resolve_history_path


@dataclass
class SessionRecord:
    timestamp: str
    subcommand: str
    agent: str
    prompt_preview: str
    success: bool
    exit_code: int
    latency_ms: float
    dry_run: bool
    metadata: Dict[str, Any]


class SessionManager:
    """Persist session information for auditability."""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.history_path = resolve_history_path(config)
        self.history_path.parent.mkdir(parents=True, exist_ok=True)

    def log(self, record: SessionRecord) -> None:
        payload = asdict(record)
        payload["timestamp"] = record.timestamp
        with self.history_path.open("a", encoding="utf-8") as fh:
            fh.write(json.dumps(payload))
            fh.write("\n")

    @staticmethod
    def create_record(
        subcommand: str,
        agent: str,
        prompt: str,
        success: bool,
        exit_code: int,
        latency_ms: float,
        dry_run: bool,
        metadata: Dict[str, Any],
    ) -> SessionRecord:
        preview = (prompt[:200] + "...") if len(prompt) > 200 else prompt
        return SessionRecord(
            timestamp=datetime.utcnow().isoformat(timespec="seconds"),
            subcommand=subcommand,
            agent=agent,
            prompt_preview=preview,
            success=success,
            exit_code=exit_code,
            latency_ms=latency_ms,
            dry_run=dry_run,
            metadata=metadata,
        )
