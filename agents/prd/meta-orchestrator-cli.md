Meta-Orchestrator CLI PRD
=========================

Overview
--------
Build a command line orchestrator that acts as a middleman across Codex CLI, Gemini CLI, Copilot CLI, and Cloud CLI. The tool normalizes invocation, credential handling, prompt routing, and result aggregation so a single entry point can dispatch work to specialized agents.

Problem Statement
-----------------
- Developers juggle multiple AI CLIs and need to remember unique syntaxes, configuration files, and capabilities.
- Switching between tools interrupts flow and introduces context duplication.
- There is no unified logging or traceability for how requests were routed across agents.

Product Goals
-------------
- Provide one CLI (`agentx`) that accepts a prompt or task file and determines which downstream agent should execute it.
- Minimize setup friction: install once, auto-detect installed CLIs, and bootstrap missing config where possible.
- Deliver consistent feedback with consolidated histories, evaluation scores, and error reporting.

Non-Goals
---------
- Building new LLM capabilities; the orchestrator reuses existing CLIs.
- Replacing proprietary UIs provided by vendors.
- Long-running workflow automation beyond synchronous request-response.

Personas
--------
- **Full-stack developer:** Wants to leverage the best agent for each task without context switching.
- **DevOps engineer:** Needs audit trails for AI-generated infrastructure scripts.
- **Team lead:** Cares about policy compliance and prefers consistent usage reporting.

User Stories
------------
1. As a developer, I run `agentx ask "Optimize this SQL query"` and the tool forwards to the best SQL-capable agent, returning output inline.
2. As an engineer, I pass a code diff file and the tool routes review tasks to Copilot CLI while logging metadata locally.
3. As a team lead, I export usage analytics to check which agents handled which requests last sprint.

Functional Requirements
-----------------------
- Command router with configurable rules (regex, heuristics, manual overrides).
- Adapter plugins for Codex CLI, Gemini CLI, Copilot CLI, and Cloud CLI:
  - Detect installation paths.
  - Normalize authentication (env vars, tokens) via secrets manager.
  - Translate incoming requests into each agent's CLI syntax.
- Session manager tracking prompts, responses, exit codes, and agent choice.
- Logging subsystem with pluggable outputs (local file, stdout, HTTP webhook).
- Config file (`~/.agentx/config.yaml`) supporting:
  - Routing rules, scoring weights, and manual pinning.
  - Rate limits and concurrency controls.
  - Feature toggles per agent.
- Source all executable and helper scripts from `scripts/agentx/`, keeping adapter, routing, and tooling code in clearly named subdirectories.
- Dry-run mode to show routing decisions without executing downstream CLIs.

Non-Functional Requirements
---------------------------
- Cross-platform support: macOS, Linux.
- CLI response within 250ms overhead beyond downstream agent runtime.
- Observability: structured logs, optional debug verbosity.
- Security: never persist raw credentials; integrate with OS keychain when possible.

User Experience & Interaction
-----------------------------
- Familiar CLI syntax with subcommands (`ask`, `review`, `plan`, `config`).
- Rich terminal output:
  - Summary header showing chosen agent, latency, confidence score.
  - Collapsible sections (via pager integration) for detailed transcripts.
- Autocomplete support for zsh and bash.
- Config wizard to help first-time setup detect existing CLIs and tokens.

Architecture Overview
---------------------
```
┌─────────────────┐
│  agentx CLI     │
├─────────────────┤
│ Command Parser  │
│ Routing Engine  │
│ Session Manager │
│ Logging Layer   │
└──┬────────────┬─┘
   │            │
┌──▼───┐   ┌────▼───┐
│Adapters│ │Config  │
│(per CLI│ │Store   │
└──┬─────┘ └────────┘
   │
┌──▼─────────────────────────────┐
│ Downstream CLIs (Codex/Gemini/ │
│ Copilot/Cloud) via subprocess  │
└────────────────────────────────┘
```

Open Questions
--------------
- Should routing heuristics be deterministic or include feedback loops (e.g., reinforcement from user ratings)?
- How to sandbox prompts to avoid leaking sensitive data across vendors?
- Should the orchestrator cache responses for identical prompts to save usage costs?

Success Metrics
---------------
- 90% of routed commands execute successfully without manual override in beta.
- Reduce average tool-switching time by 60% (self-reported).
- Achieve a 4.5/5 satisfaction score from pilot users on usability surveys.

Milestones
----------
1. Prototype: detect installed CLIs, execute pass-through requests (2 weeks).
2. Routing MVP: keyword heuristics, logging, config file (3 weeks).
3. Beta release: advanced rules, analytics export, setup wizard (4 weeks).
4. GA: plugin marketplace support, extended documentation (2 weeks).

Risks & Mitigations
-------------------
- **Inconsistent CLI outputs:** Normalize with JSON adapters; add tests for known prompts.
- **Credential mishandling:** Leverage OS keychain APIs; require explicit opt-in to store tokens.
- **Downstream CLI changes:** Version compatibility checks and adapter update alerts.

Dependencies
------------
- Access to installed CLIs and their authentication tokens.
- OS keychain / secret management libraries.
- Logging backend (local file system by default).

Appendix
--------
- Potential future feature: interactive TUI layer for browsing history.
- Consider extension API so third-party agents can register themselves.
