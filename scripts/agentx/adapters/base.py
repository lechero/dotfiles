"""Base adapter class for downstream CLIs."""
from __future__ import annotations

import os
import shlex
import shutil
import subprocess
import time
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Sequence


@dataclass
class AdapterRequest:
    subcommand: str
    prompt: str
    arguments: Sequence[str] = field(default_factory=list)
    context_files: Sequence[str] = field(default_factory=list)
    metadata: Dict[str, object] = field(default_factory=dict)
    dry_run: bool = False
    extra_env: Dict[str, str] = field(default_factory=dict)


@dataclass
class AdapterExecutionResult:
    success: bool
    exit_code: int
    output: str
    error: Optional[str]
    duration_ms: float
    command: str
    adapter: str
    metadata: Dict[str, object] = field(default_factory=dict)


class BaseAdapter:
    """Common adapter logic for downstream CLIs."""

    name = "base"
    display_name = "Base"
    capability_tags: Sequence[str] = ()
    DEFAULT_COMMAND_PREFIX: Sequence[str] = ()
    SUBCOMMAND_MAP: Dict[str, Sequence[str]] = {}

    def __init__(self, config: Dict[str, object], logger):
        self.config = config or {}
        self.logger = logger
        self._resolved_prefix: Optional[List[str]] = None

    # ------------------------------------------------------------------
    # Detection helpers
    # ------------------------------------------------------------------
    def is_available(self) -> bool:
        return self.resolve_command_prefix() is not None

    def resolve_command_prefix(self) -> Optional[List[str]]:
        if self._resolved_prefix is not None:
            return self._resolved_prefix
        prefix = list(self.config.get("command_prefix", self.DEFAULT_COMMAND_PREFIX))
        if not prefix:
            return None
        binary_path = shutil.which(prefix[0])
        if binary_path:
            resolved = [binary_path] + prefix[1:]
            self._resolved_prefix = resolved
            return resolved
        self.logger.debug(
            "Binary not found for adapter",
            extra={"extra_data": {"adapter": self.name, "binary": prefix[0]}},
        )
        return None

    # ------------------------------------------------------------------
    # Execution
    # ------------------------------------------------------------------
    def execute(self, request: AdapterRequest) -> AdapterExecutionResult:
        command_prefix = self.resolve_command_prefix()
        if command_prefix is None:
            return AdapterExecutionResult(
                success=False,
                exit_code=1,
                output="",
                error=f"{self.display_name} CLI not found on PATH. Configure 'command_prefix' in config.yaml.",
                duration_ms=0.0,
                command="",
                adapter=self.name,
            )

        command = self.build_command(command_prefix, request)
        prompt_delivery = self.config.get("prompt_via", "stdin")
        env = os.environ.copy()
        env.update(request.extra_env)

        if prompt_delivery == "argument":
            prompt_arg = self.config.get("prompt_arg", "--prompt")
            command = command + [prompt_arg, request.prompt]
            input_data = None
        elif prompt_delivery == "env":
            env_var = self.config.get("prompt_env_var", "AGENTX_PROMPT")
            env[env_var] = request.prompt
            input_data = None
        else:  # stdin
            input_data = request.prompt

        rendered_command = " ".join(shlex.quote(part) for part in command)

        if request.dry_run:
            return AdapterExecutionResult(
                success=True,
                exit_code=0,
                output=f"DRY RUN: would execute {rendered_command}",
                error=None,
                duration_ms=0.0,
                command=rendered_command,
                adapter=self.name,
                metadata={"prompt_via": prompt_delivery},
            )

        start = time.time()
        try:
            completed = subprocess.run(  # noqa: S603, S607
                command,
                input=input_data,
                text=True,
                capture_output=True,
                check=False,
                env=env,
            )
        except OSError as exc:  # pragma: no cover - depends on system
            self.logger.error(
                "Failed to invoke adapter",
                extra={"extra_data": {"adapter": self.name, "error": str(exc)}},
            )
            return AdapterExecutionResult(
                success=False,
                exit_code=1,
                output="",
                error=str(exc),
                duration_ms=(time.time() - start) * 1000,
                command=rendered_command,
                adapter=self.name,
            )

        duration_ms = (time.time() - start) * 1000
        success = completed.returncode == 0
        error_text = completed.stderr.strip() if completed.stderr else None
        output_text = completed.stdout.strip()
        return AdapterExecutionResult(
            success=success,
            exit_code=completed.returncode,
            output=output_text,
            error=error_text,
            duration_ms=duration_ms,
            command=rendered_command,
            adapter=self.name,
            metadata={"prompt_via": prompt_delivery},
        )

    def build_command(self, command_prefix: List[str], request: AdapterRequest) -> List[str]:
        extras = list(self.SUBCOMMAND_MAP.get(request.subcommand, []))
        extras.extend(request.arguments)
        return command_prefix + extras
