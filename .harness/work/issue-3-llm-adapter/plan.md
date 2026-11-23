# Plan: Issue #3 — Decoupled Gemini Adapter & Output Validation

## Affected Topics
- `llm-adapter`
- `domain-core`

## Outcomes (WHAT)
1. **Provider-Decoupled Gemini Adapter**: Implements `touchpoint.LLMPort` (`GenerateMicroDrill`, `EvaluateAnswer`) and `learning.EmbeddingPort` (`EmbedChunks`) inside `internal/adapters/llm/gemini/`.
2. **Strict Grounding & Friction Adaptation**: Rejects ungrounded requests (empty corpus) and adapts drill format to `MORNING_CATALYST` (<200 words) vs `EVENING_STREAK_SAVER` (<=60 words).
3. **Deterministic Output Guardrails**: Validates JSON structure, word counts, score bounds `[0.0, 1.0]`, and `Passed` threshold consistency with bounded corrective retry.
4. **Offline Contract Tests**: Full unit and recorded-response test suite executes deterministically without API keys.

## Non-Goals
- OpenAI / Anthropic / Ollama adapters (architecture supports them, implementation deferred).
- Multi-path synthesis & path affinity classification (scoped to Issue #6).
- Live network integration tests in CI.
