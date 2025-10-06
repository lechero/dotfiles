"""Configuration utilities for the AgentX CLI."""
from __future__ import annotations

import copy
import json
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Optional

try:
    import yaml  # type: ignore
except ModuleNotFoundError:  # pragma: no cover - optional dependency
    yaml = None  # type: ignore

CONFIG_DIR = Path.home() / ".agentx"
CONFIG_PATH = CONFIG_DIR / "config.yaml"
HISTORY_PATH = CONFIG_DIR / "history.jsonl"
LOG_PATH = CONFIG_DIR / "agentx.log"


DEFAULT_CONFIG: Dict[str, Any] = {
    "routing": {
        "overrides": {},  # {"ask": "codex", "review": "copilot"}
        "keywords": {
            "sql": "codex",
            "terraform": "cloud",
            "kubernetes": "cloud",
            "k8s": "cloud",
            "design": "gemini",
            "architecture": "gemini",
            "pull request": "copilot",
            "review": "copilot",
        },
        "weights": {
            "codex": 0.6,
            "gemini": 0.6,
            "copilot": 0.7,
            "cloud": 0.65,
        },
    },
    "adapters": {
        "codex": {
            "command_prefix": ["codex"],
            "prompt_via": "stdin",
        },
        "gemini": {
            "command_prefix": ["gemini"],
            "prompt_via": "stdin",
        },
        "copilot": {
            "command_prefix": ["gh", "copilot"],
            "prompt_via": "stdin",
        },
        "cloud": {
            "command_prefix": ["gcloud", "ai", "models", "predict"],
            "prompt_via": "stdin",
        },
    },
    "logging": {
        "level": "INFO",
        "output": "stdout",
        "file_path": str(LOG_PATH),
        "format": "json",
    },
    "session": {
        "history_path": str(HISTORY_PATH),
    },
    "features": {
        "autocomplete": True,
        "analytics_export": True,
        "dry_run_default": False,
    },
}


@dataclass
class ConfigLoadResult:
    path: Path
    data: Dict[str, Any]
    created: bool = False


def ensure_config_dir() -> None:
    """Ensure that the AgentX configuration directory exists."""
    CONFIG_DIR.mkdir(mode=0o700, parents=True, exist_ok=True)


def _dump_yaml(data: Dict[str, Any]) -> str:
    """Serialize configuration data to YAML (with JSON fallback)."""
    if yaml is not None:
        return yaml.safe_dump(data, sort_keys=False)
    return json.dumps(data, indent=2)


def _load_yaml(text: str) -> Dict[str, Any]:
    if not text.strip():
        return {}
    if yaml is not None:
        return yaml.safe_load(text) or {}
    return json.loads(text)


def _deep_merge(base: Dict[str, Any], override: Dict[str, Any]) -> Dict[str, Any]:
    result = copy.deepcopy(base)
    for key, value in override.items():
        if isinstance(value, dict) and isinstance(result.get(key), dict):
            result[key] = _deep_merge(result[key], value)
        else:
            result[key] = value
    return result


def load_config(create_missing: bool = True) -> ConfigLoadResult:
    """Load the AgentX configuration file, creating defaults when needed."""
    ensure_config_dir()
    created = False
    if not CONFIG_PATH.exists():
        if create_missing:
            save_config(DEFAULT_CONFIG)
            created = True
        else:
            return ConfigLoadResult(path=CONFIG_PATH, data={}, created=False)

    with CONFIG_PATH.open("r", encoding="utf-8") as fh:
        raw = fh.read()
    user_config = _load_yaml(raw)
    data = _deep_merge(DEFAULT_CONFIG, user_config)
    return ConfigLoadResult(path=CONFIG_PATH, data=data, created=created)


def save_config(data: Dict[str, Any]) -> None:
    """Persist configuration data to disk."""
    ensure_config_dir()
    serialized = _dump_yaml(data)
    with CONFIG_PATH.open("w", encoding="utf-8") as fh:
        fh.write(serialized)


def config_exists() -> bool:
    return CONFIG_PATH.exists()


def resolve_history_path(config: Optional[Dict[str, Any]] = None) -> Path:
    cfg = config or DEFAULT_CONFIG
    path = Path(cfg.get("session", {}).get("history_path", str(HISTORY_PATH)))
    return Path(os.path.expanduser(path))


def resolve_log_path(config: Optional[Dict[str, Any]] = None) -> Path:
    cfg = config or DEFAULT_CONFIG
    path = Path(cfg.get("logging", {}).get("file_path", str(LOG_PATH)))
    return Path(os.path.expanduser(path))
