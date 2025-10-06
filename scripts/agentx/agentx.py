"""Entry point for the AgentX meta-orchestrator CLI."""
from __future__ import annotations

import argparse
import csv
import io
import json
import sys
import textwrap
from pathlib import Path
from typing import Dict, Iterable, List, Tuple

from . import config as config_module
from .adapters import (
    AdapterExecutionResult,
    AdapterRequest,
    BaseAdapter,
    CloudAdapter,
    CodexAdapter,
    CopilotAdapter,
    GeminiAdapter,
)
from .cli import build_parser
from .logging_utils import configure_logging, emit_startup_banner
from .routing import RoutingDecision, RoutingEngine
from .session import SessionManager

ADAPTER_REGISTRY = {
    "codex": CodexAdapter,
    "gemini": GeminiAdapter,
    "copilot": CopilotAdapter,
    "cloud": CloudAdapter,
}


def main(argv: Iterable[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    if args.config_path:
        override_config_path(Path(args.config_path))

    config_result = config_module.load_config()

    if args.verbose:
        config_result.data.setdefault("logging", {})["level"] = "DEBUG"

    logger = configure_logging(config_result.data)
    emit_startup_banner(logger, config_result.data)

    adapters, availability = instantiate_adapters(config_result.data, logger)
    installed_agents = [name for name, available in availability.items() if available]
    routing_agents = installed_agents or list(adapters.keys())
    parser = build_parser(adapters.keys())  # refresh parser with agent choices for --agent help text
    args = parser.parse_args(argv)

    session_manager = SessionManager(config_result.data)

    if args.subcommand == "config":
        return handle_config_command(args, config_result.data)
    if args.subcommand == "export":
        return handle_export_command(args, config_result.data)

    prompt = resolve_prompt(args)
    dry_run = bool(args.dry_run or config_result.data.get("features", {}).get("dry_run_default"))
    context_files = resolve_context_files(args)

    target_adapter_name: str
    routing_decision: RoutingDecision
    if getattr(args, "agent", None):
        target_adapter_name = args.agent
        routing_decision = RoutingDecision(
            agent=target_adapter_name,
            confidence=1.0,
            reason="Manual selection via --agent",
            scores={target_adapter_name: 1.0},
            manual_override=True,
        )
    else:
        router = RoutingEngine(config_result.data, routing_agents)
        routing_decision = router.route(
            subcommand=args.subcommand,
            prompt=prompt,
            metadata={"context_files": context_files},
        )
        target_adapter_name = routing_decision.agent

    adapter = adapters.get(target_adapter_name)
    if adapter is None:
        print(f"Unknown adapter '{target_adapter_name}'.", file=sys.stderr)
        return 2

    if not availability.get(target_adapter_name) and not dry_run:
        print(
            textwrap.dedent(
                f"""
                Adapter '{target_adapter_name}' is not currently available on PATH.
                Update ~/.agentx/config.yaml with the correct command_prefix or enable --dry-run.
                """
            ).strip(),
            file=sys.stderr,
        )

    request = AdapterRequest(
        subcommand=args.subcommand,
        prompt=prompt,
        arguments=tuple(),
        context_files=context_files,
        metadata={"routing": routing_decision.to_dict()},
        dry_run=dry_run,
    )
    result = adapter.execute(request)

    render_summary(routing_decision, result)

    session_record = SessionManager.create_record(
        subcommand=args.subcommand,
        agent=target_adapter_name,
        prompt=prompt,
        success=result.success,
        exit_code=result.exit_code,
        latency_ms=result.duration_ms,
        dry_run=dry_run,
        metadata={"routing": routing_decision.to_dict()},
    )
    session_manager.log(session_record)

    if result.success:
        if result.output:
            print(result.output)
        return result.exit_code

    if result.error:
        print(result.error, file=sys.stderr)
    return result.exit_code or 1


def instantiate_adapters(config: Dict[str, object], logger) -> Tuple[Dict[str, BaseAdapter], Dict[str, bool]]:
    adapters: Dict[str, BaseAdapter] = {}
    availability: Dict[str, bool] = {}
    adapter_cfg = config.get("adapters", {})
    for name, cls in ADAPTER_REGISTRY.items():
        cfg = adapter_cfg.get(name, {})
        adapter = cls(cfg, logger)
        adapters[name] = adapter
        availability[name] = adapter.is_available()
    return adapters, availability


def override_config_path(path: Path) -> None:
    expanded = path.expanduser().resolve()
    config_module.CONFIG_PATH = expanded
    config_module.CONFIG_DIR = expanded.parent
    config_module.HISTORY_PATH = config_module.CONFIG_DIR / "history.jsonl"
    config_module.LOG_PATH = config_module.CONFIG_DIR / "agentx.log"
    config_module.DEFAULT_CONFIG.setdefault("session", {})["history_path"] = str(config_module.HISTORY_PATH)
    config_module.DEFAULT_CONFIG.setdefault("logging", {})["file_path"] = str(config_module.LOG_PATH)


def resolve_prompt(args: argparse.Namespace) -> str:
    if getattr(args, "prompt", None):
        return args.prompt
    if getattr(args, "prompt_file", None):
        path = Path(args.prompt_file).expanduser()
        if not path.exists():
            raise SystemExit(f"Prompt file not found: {path}")
        return path.read_text(encoding="utf-8")
    raise SystemExit("Prompt text or --prompt-file is required.")


def resolve_context_files(args: argparse.Namespace) -> List[str]:
    files: List[str] = []
    for attr in ("context_file", "diff_file"):
        value = getattr(args, attr, None)
        if not value:
            continue
        if isinstance(value, list):
            files.extend(str(Path(item).expanduser()) for item in value)
        else:
            files.append(str(Path(value).expanduser()))
    return files


def render_summary(decision: RoutingDecision, result: AdapterExecutionResult) -> None:
    header = f"agent={decision.agent} confidence={decision.confidence:.2f} override={decision.manual_override}"
    print(header)
    if decision.reason:
        print(f"reason={decision.reason}")
    if decision.scores:
        ranked = ", ".join(
            f"{agent}:{score:.2f}" for agent, score in sorted(decision.scores.items(), key=lambda kv: kv[1], reverse=True)
        )
        print(f"scores={ranked}")
    print(f"command={result.command or 'n/a'}")
    print(f"duration_ms={result.duration_ms:.1f}")


def handle_config_command(args: argparse.Namespace, config: Dict[str, object]) -> int:
    if args.path:
        print(config_module.CONFIG_PATH)
        return 0
    if args.init:
        if config_module.config_exists():
            print("Configuration already exists.")
            return 0
        config_module.save_config(config_module.DEFAULT_CONFIG)
        print(f"Default configuration created at {config_module.CONFIG_PATH}")
        return 0
    if args.show:
        try:
            serialized = config_module._dump_yaml(config)  # type: ignore[attr-defined]
        except AttributeError:
            serialized = json.dumps(config, indent=2)
        print(serialized)
        return 0
    parser = build_parser()
    parser.print_help()
    return 0


def handle_export_command(args: argparse.Namespace, config: Dict[str, object]) -> int:
    history_path = config_module.resolve_history_path(config)
    if not history_path.exists():
        print("No session history found.")
        return 0

    with history_path.open("r", encoding="utf-8") as fh:
        entries = [json.loads(line) for line in fh if line.strip()]

    if args.format == "jsonl":
        payload = "\n".join(json.dumps(entry) for entry in entries)
    else:  # csv
        if not entries:
            payload = ""
        else:
            fieldnames = sorted(entries[0].keys())
            buffer = io.StringIO()
            writer = csv.DictWriter(buffer, fieldnames=fieldnames)
            writer.writeheader()
            for entry in entries:
                writer.writerow(entry)
            payload = buffer.getvalue()

    if args.output:
        Path(args.output).expanduser().write_text(payload, encoding="utf-8")
    else:
        print(payload)
    return 0


if __name__ == "__main__":
    sys.exit(main())
