"""Cloud CLI adapter."""
from __future__ import annotations

from typing import Sequence

from .base import BaseAdapter


class CloudAdapter(BaseAdapter):
    name = "cloud"
    display_name = "Cloud CLI"
    capability_tags: Sequence[str] = ("devops", "infrastructure", "cloud")
    DEFAULT_COMMAND_PREFIX: Sequence[str] = ("gcloud", "ai", "models", "predict")
    SUBCOMMAND_MAP = {
        "ask": tuple(),
        "plan": tuple(),
        "review": tuple(),
    }
