"""Codex CLI adapter."""
from __future__ import annotations

from typing import Sequence

from .base import BaseAdapter


class CodexAdapter(BaseAdapter):
    name = "codex"
    display_name = "Codex CLI"
    capability_tags: Sequence[str] = ("codegen", "analysis", "sql")
    DEFAULT_COMMAND_PREFIX: Sequence[str] = ("codex",)
    SUBCOMMAND_MAP = {
        "ask": ("ask",),
        "review": ("review",),
        "plan": ("plan",),
    }
