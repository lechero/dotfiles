Agent Router Daemon PRD
=======================

Overview
--------
Deliver a long-running background service that exposes a local API for orchestrating Codex CLI, Gemini CLI, Copilot CLI, and Cloud CLI. The daemon accepts HTTP or gRPC requests from any client (CLI, GUI, scripts) and manages queued jobs, context sharing, and policy enforcement before delegating to downstream agents.

Problem Statement
-----------------
- Teams want to embed AI agent orchestration into IDE extensions, CI pipelines, and chat bots, but current CLIs are strictly interactive.
- Running multiple CLIs concurrently is resource intensive and hard to monitor.
- There is no centralized controller to apply organizational policies (rate limits, vendor preferences, compliance filters).

Product Goals
-------------
- Provide a persistent daemon (`agent-routerd`) reachable via local network interfaces.
- Support REST and optional gRPC endpoints for submitting tasks, checking status, and retrieving results.
- Offer a rule engine for routing, queuing, and multiplexing requests across available CLIs.
- Enable organization-wide policy enforcement and observability.

Non-Goals
---------
- Building cloud-hosted orchestration; the daemon runs locally or within private infrastructure.
- Replacing vendor account management or billing dashboards.
- Providing human-in-the-loop review workflows (future consideration).

Personas
--------
- **Platform engineer:** Embeds the daemon in internal developer platforms and pipelines.
- **Security officer:** Needs centralized auditing and policy controls.
- **Automation engineer:** Triggers agent tasks programmatically from scripts or bots.

User Stories
------------
1. As a platform engineer, I deploy the daemon on a developer workstation and use REST to submit prompts coming from a VS Code extension.
2. As a security officer, I configure policies that block routing to external vendors when code contains classified keywords.
3. As an automation engineer, I enqueue 100 code refactoring requests; the daemon batches them across available GPUs and returns job IDs for later retrieval.

Functional Requirements
-----------------------
- API endpoints:
  - `POST /tasks`: submit a task with payload, desired capabilities, and priority.
  - `GET /tasks/{id}`: retrieve status, transcript, and outcomes.
  - `POST /policies`: define routing and compliance rules.
  - Websocket or SSE endpoint for streaming responses.
- Organize daemon executables, worker scripts, and admin tooling inside `scripts/agent-routerd/` with descriptive subfolders (e.g., `workers/`, `ctl/`).
- Job scheduler with priority queues, cancellation, and retry semantics.
- Policy engine supporting declarative rules (YAML) evaluating payload metadata and content classifiers.
- Adapter layer for each CLI:
  - Pooled workers executing downstream commands.
  - Structured capture of stdout/stderr, exit codes, and tokens usage.
- State persistence: embedded SQLite or equivalent to store tasks, logs, metrics.
- Admin dashboard (optional first-party CLI) to inspect queue status and active workers.

Non-Functional Requirements
---------------------------
- Concurrency: handle 100 concurrent tasks with graceful degradation.
- Latency: daemon adds less than 500ms scheduling overhead.
- Security: authentication via API keys or mutual TLS; encrypted persistence.
- Observability: metrics endpoints (Prometheus), structured logs, tracing hooks.

Experience & Interaction
------------------------
- Provide a companion CLI (`agent-routerctl`) for submitting tasks, watching logs, and managing policies.
- Offer SDK snippets (Python, Node.js) to simplify integration with scripts and bots.
- Supply policy templates for common use cases (e.g., route coding tasks to Codex, data analysis to Gemini).

Architecture Overview
---------------------
```
┌────────────┐   REST/gRPC   ┌───────────────────┐
│ Clients    │──────────────▶│ Agent Router Daemon│
│ (CLI/UI)   │               ├───────────────────┤
└────────────┘               │ API Layer         │
                             │ Auth & Policies   │
                             │ Scheduler         │
                             │ Persistence       │
                             └──┬─────────────┬──┘
                                │             │
                      ┌─────────▼──┐   ┌──────▼─────┐
                      │CLI Workers │   │Metrics/Logs│
                      │(Codex etc.)│   │+ Dashboards│
                      └────────────┘   └────────────┘
```

Success Metrics
---------------
- 95% of tasks complete within expected SLA for beta customers.
- 80% reduction in manual CLI orchestration steps reported by pilot teams.
- Zero critical security incidents in first six months of deployment.

Milestones
----------
1. Core daemon skeleton with REST endpoints and single worker adapter (3 weeks).
2. Scheduler + policy engine MVP with two CLI adapters (4 weeks).
3. Persistence, metrics, and SDK release (3 weeks).
4. Enterprise hardening: auth, encryption, policy pack (4 weeks).

Risks & Mitigations
-------------------
- **Resource exhaustion:** Implement worker pooling and back-pressure controls.
- **Policy misconfiguration:** Provide validation tooling and dry-run simulations.
- **API attack surface:** Harden with rate limiting, authentication, and secure defaults.

Dependencies
------------
- Access to local or containerized instances of required CLIs.
- Content classification libraries (e.g., open-source DLP filters) for policy evaluation.
- Prometheus-compatible metrics stack for observability.

Open Questions
--------------
- Should the daemon support remote worker nodes for distributed execution?
- How to price or license the daemon if it transitions from internal to commercial product?
- What is the best strategy for storing potentially sensitive prompt/response data?
