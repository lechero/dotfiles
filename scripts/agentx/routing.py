"""Routing engine for the AgentX orchestrator."""
from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import Dict, Iterable, List, Optional


@dataclass
class RoutingDecision:
    agent: str
    confidence: float
    reason: str
    scores: Dict[str, float] = field(default_factory=dict)
    manual_override: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "agent": self.agent,
            "confidence": round(self.confidence, 2),
            "reason": self.reason,
            "manual_override": self.manual_override,
            "scores": {k: round(v, 3) for k, v in sorted(self.scores.items(), key=lambda kv: kv[1], reverse=True)},
        }


class RoutingEngine:
    """Determine which downstream agent should handle a request."""

    def __init__(self, config: Dict[str, object], available_agents: Iterable[str]):
        self.config = config
        self.available_agents = set(available_agents)

    def route(self, subcommand: str, prompt: str, metadata: Optional[Dict[str, object]] = None) -> RoutingDecision:
        routing_cfg = self.config.get("routing", {})
        overrides = routing_cfg.get("overrides", {})
        manual_agent = overrides.get(subcommand)
        if manual_agent:
            return RoutingDecision(
                agent=manual_agent,
                confidence=1.0,
                reason=f"Manual override defined for '{subcommand}'",
                manual_override=True,
            )

        scores: Dict[str, float] = {agent: 0.1 for agent in self.available_agents}
        scores.update({agent: 0.1 for agent in ("codex", "gemini", "copilot", "cloud")})
        keywords: Dict[str, str] = routing_cfg.get("keywords", {})
        keyword_hits: Dict[str, int] = {agent: 0 for agent in scores}

        lowered_prompt = prompt.lower()
        for phrase, agent in keywords.items():
            if agent not in scores:
                continue
            pattern = re.escape(phrase.lower())
            matches = len(re.findall(pattern, lowered_prompt))
            if matches:
                keyword_hits[agent] = keyword_hits.get(agent, 0) + matches

        weights: Dict[str, float] = routing_cfg.get("weights", {})
        for agent, hits in keyword_hits.items():
            if hits:
                scores[agent] = scores.get(agent, 0.0) + hits * weights.get(agent, 0.5)

        # Subcommand defaults provide a gentle nudge.
        subcommand_defaults = {
            "ask": "codex",
            "review": "copilot",
            "plan": "gemini",
        }
        if subcommand in subcommand_defaults:
            default_agent = subcommand_defaults[subcommand]
            scores[default_agent] = scores.get(default_agent, 0.0) + 0.2

        # Context metadata heuristics.
        metadata = metadata or {}
        context_files = metadata.get("context_files", [])
        for path in context_files:
            suffix = str(path).lower()
            if suffix.endswith(".tf"):
                scores["cloud"] = scores.get("cloud", 0.0) + 0.5
            elif suffix.endswith(".sql"):
                scores["codex"] = scores.get("codex", 0.0) + 0.5
            elif suffix.endswith((".md", ".pptx")):
                scores["gemini"] = scores.get("gemini", 0.0) + 0.3

        # Pick the best available agent.
        sorted_scores = sorted(scores.items(), key=lambda kv: kv[1], reverse=True)
        for agent, score in sorted_scores:
            if agent in self.available_agents:
                confidence = min(1.0, max(0.05, score))
                reason = self._build_reason(agent, score, keyword_hits, subcommand_defaults.get(subcommand))
                return RoutingDecision(agent=agent, confidence=confidence, reason=reason, scores=scores)

        # Fall back to the first declared adapter.
        fallback_agent = next(iter(self.available_agents))
        return RoutingDecision(
            agent=fallback_agent,
            confidence=0.2,
            reason="No heuristic matched; defaulting to first available adapter",
            scores=scores,
        )

    @staticmethod
    def _build_reason(agent: str, score: float, keyword_hits: Dict[str, int], default_agent: Optional[str]) -> str:
        reasons: List[str] = []
        if keyword_hits.get(agent):
            reasons.append(f"{keyword_hits[agent]} keyword hit(s)")
        if default_agent == agent:
            reasons.append("default for subcommand")
        if score >= 1.0:
            reasons.append("strong score")
        if not reasons:
            reasons.append("highest heuristic score")
        return ", ".join(reasons)
