"""Gemini CLI adapter."""
from __future__ import annotations

from typing import Sequence

from .base import BaseAdapter


class GeminiAdapter(BaseAdapter):
    name = "gemini"
    display_name = "Gemini CLI"
    capability_tags: Sequence[str] = ("analysis", "planning", "design")
    DEFAULT_COMMAND_PREFIX: Sequence[str] = ("gemini",)
    SUBCOMMAND_MAP = {
        "ask": ("prompt",),
        "plan": ("plan",),
    }
