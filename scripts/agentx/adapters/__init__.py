"""Adapter implementations for AgentX."""
from .base import AdapterExecutionResult, AdapterRequest, BaseAdapter
from .cloud import CloudAdapter
from .codex import CodexAdapter
from .copilot import CopilotAdapter
from .gemini import GeminiAdapter

__all__ = [
    "AdapterExecutionResult",
    "AdapterRequest",
    "BaseAdapter",
    "CloudAdapter",
    "CodexAdapter",
    "CopilotAdapter",
    "GeminiAdapter",
]
