"""GitHub Copilot CLI adapter."""
from __future__ import annotations

from typing import Sequence

from .base import BaseAdapter


class CopilotAdapter(BaseAdapter):
    name = "copilot"
    display_name = "GitHub Copilot CLI"
    capability_tags: Sequence[str] = ("review", "explain", "tests")
    DEFAULT_COMMAND_PREFIX: Sequence[str] = ("gh", "copilot")
    SUBCOMMAND_MAP = {
        "ask": ("chat",),
        "review": ("review",),
        "plan": ("summarize",),
    }
