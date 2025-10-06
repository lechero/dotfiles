"""Command-line interface definition for AgentX."""
from __future__ import annotations

import argparse
from typing import Iterable, Optional


def build_parser(available_agents: Optional[Iterable[str]] = None) -> argparse.ArgumentParser:
    agents = sorted(set(available_agents or []))

    parser = argparse.ArgumentParser(
        prog="agentx",
        description="Meta-orchestrator CLI that routes prompts to downstream AI CLIs.",
    )
    parser.add_argument(
        "--config",
        dest="config_path",
        help="Optional path to an alternate config.yaml file.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show routing decision and adapter command without executing it.",
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Increase logging verbosity for debugging.",
    )

    subparsers = parser.add_subparsers(dest="subcommand", required=True)

    def add_prompt_arguments(sub_parser: argparse.ArgumentParser) -> None:
        sub_parser.add_argument(
            "prompt",
            nargs="?",
            help="Prompt text. If omitted, provide --prompt-file.",
        )
        sub_parser.add_argument(
            "--prompt-file",
            help="Path to a file containing the prompt.",
        )
        sub_parser.add_argument(
            "--context-file",
            action="append",
            default=[],
            help="Additional file(s) to provide as context for routing heuristics.",
        )
        if agents:
            sub_parser.add_argument(
                "--agent",
                choices=agents,
                help="Force a specific downstream agent.",
            )
        else:
            sub_parser.add_argument(
                "--agent",
                help="Force a specific downstream agent.",
            )

    ask_parser = subparsers.add_parser("ask", help="Send a general prompt to the orchestrator.")
    add_prompt_arguments(ask_parser)

    review_parser = subparsers.add_parser("review", help="Route a review or critique task.")
    add_prompt_arguments(review_parser)
    review_parser.add_argument(
        "--diff-file",
        help="Optional diff file to include in the review context.",
    )

    plan_parser = subparsers.add_parser("plan", help="Generate plans, designs, or architecture guidance.")
    add_prompt_arguments(plan_parser)

    config_parser = subparsers.add_parser("config", help="Inspect or update orchestrator configuration.")
    config_actions = config_parser.add_mutually_exclusive_group()
    config_actions.add_argument("--show", action="store_true", help="Print the effective configuration.")
    config_actions.add_argument(
        "--init",
        action="store_true",
        help="Create the default configuration if it does not exist.",
    )
    config_actions.add_argument(
        "--path",
        action="store_true",
        help="Print the path to the configuration file.",
    )

    export_parser = subparsers.add_parser(
        "export",
        help="Export session history for analytics.",
    )
    export_parser.add_argument(
        "--format",
        choices=["jsonl", "csv"],
        default="jsonl",
        help="Format of the exported session history.",
    )
    export_parser.add_argument(
        "--output",
        help="Output file path. Defaults to stdout.",
    )

    return parser
