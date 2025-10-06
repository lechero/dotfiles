"""Logging helpers for the AgentX CLI."""
from __future__ import annotations

import json
import logging
import sys
from pathlib import Path
from typing import Any, Dict

from . import config as config_module


def configure_logging(cfg: Dict[str, Any]) -> logging.Logger:
    """Configure a logger according to CLI settings."""
    log_cfg = cfg.get("logging", {})
    level = getattr(logging, log_cfg.get("level", "INFO").upper(), logging.INFO)
    formatter_style = log_cfg.get("format", "json")
    logger = logging.getLogger("agentx")
    logger.setLevel(level)

    # Clear existing handlers to avoid duplicates on repeated invocations.
    logger.handlers.clear()

    handlers = []
    if log_cfg.get("output", "stdout") == "file":
        path = config_module.resolve_log_path(cfg)
        path.parent.mkdir(parents=True, exist_ok=True)
        file_handler = logging.FileHandler(path)
        handlers.append(file_handler)
    else:
        handlers.append(logging.StreamHandler(sys.stdout))

    for handler in handlers:
        if formatter_style == "json":
            handler.setFormatter(JsonLogFormatter())
        else:
            formatter = logging.Formatter(
                fmt="[%(levelname)s] %(asctime)s %(name)s: %(message)s",
                datefmt="%Y-%m-%dT%H:%M:%S",
            )
            handler.setFormatter(formatter)
        logger.addHandler(handler)

    logger.debug("Logging configured", extra={"config": log_cfg})
    return logger


class JsonLogFormatter(logging.Formatter):
    """A minimal JSON log formatter for structured output."""

    def format(self, record: logging.LogRecord) -> str:  # noqa: D401
        payload = {
            "level": record.levelname,
            "name": record.name,
            "time": self.formatTime(record, "%Y-%m-%dT%H:%M:%S"),
            "message": record.getMessage(),
        }
        if record.exc_info:
            payload["error"] = self.formatException(record.exc_info)
        if record.__dict__.get("extra_data"):
            payload["extra"] = record.__dict__["extra_data"]
        return json.dumps(payload)


def emit_startup_banner(logger: logging.Logger, cfg: Dict[str, Any]) -> None:
    logger.info(
        "AgentX orchestrator ready",
        extra={
            "extra_data": {
                "routing_overrides": list(cfg.get("routing", {}).get("overrides", {}).keys()),
                "log_output": cfg.get("logging", {}).get("output", "stdout"),
            }
        },
    )
