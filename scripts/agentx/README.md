# AgentX Meta-Orchestrator CLI

AgentX provides a single command-line entry point that routes prompts to specialized AI CLIs (Codex, Gemini, GitHub Copilot, Cloud). It implements routing heuristics, centralized logging, and session history, following the meta-orchestrator PRD.

## Quick Start

```
python scripts/agentx/agentx.py ask "Optimize this SQL query"
```

First run will materialize `~/.agentx/config.yaml`. Edit the `adapters` section to match local CLI installations. Use `--dry-run` to inspect routing without executing downstream tools:

```
python scripts/agentx/agentx.py ask --dry-run "Summarize this pull request"
```

## Key Features

- Subcommands: `ask`, `review`, `plan`, with `config` and `export` utilities.
- Routing engine that blends manual overrides, keyword heuristics, and context file hints.
- Session logging to JSONL (`~/.agentx/history.jsonl`) for analytics and audit trails.
- Structured logging to stdout or log file (`~/.agentx/agentx.log`).
- Adapter framework with pluggable command prefixes and prompt delivery strategies.

## Configuration

`agentx config --show` prints the merged configuration. To regenerate defaults, delete `~/.agentx/config.yaml` or run `agentx config --init` (when no file exists). Override routing decisions by editing the `routing.overrides` map:

```yaml
routing:
  overrides:
    review: copilot
```

Adapter command prefixes default to common binary names. If your environment differs, update `adapters.<name>.command_prefix`. Set `prompt_via` to `argument` and define `prompt_arg` when a CLI expects prompt text as an argument rather than stdin.

## History Export

Export usage history in JSONL (default) or CSV:

```
python scripts/agentx/agentx.py export --format csv --output agentx-history.csv
```

## Next Steps

- Implement adapter-specific response parsing and richer error handling.
- Integrate credentials via OS keychain APIs.
- Add analytics export to external sinks (HTTP/webhooks).
- Build interactive TUI surfaces and autocomplete scripts.
