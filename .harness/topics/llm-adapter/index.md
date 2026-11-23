# Topic: `llm-adapter`

## Purpose
Provider-decoupled AI intelligence adapter (`internal/adapters/llm/gemini/`) implementing `touchpoint.LLMPort` (and embedding generation) using the Google Gemini SDK, guarded by deterministic Go schema & output validation.

## Scope & Related Issue
- **Issue #3**: `feat(llm): implement decoupled gemini adapter and output validation`
- **Implements**:
  - `touchpoint.LLMPort` (`GenerateMicroDrill`, `EvaluateAnswer`)
  - `EmbeddingPort` (`EmbedChunks(ctx, chunks []string) ([][]float32, error)`)
  - Extensible for Issue #6 (`ClassifyPathAffinity`, `SynthesizeCrossPath`, `SynthesizeMilestone`)

## Architectural Invariants
1. **Dependency Inversion**: Domain packages never import Google GenAI SDKs. Swapping providers (`LLM_PROVIDER=gemini|openai|ollama`) only changes adapter wiring in `cmd/daemon`.
2. **Strict Grounding (Zero Hallucination Guardrail)**:
   - `GenerateMicroDrill` and `EvaluateAnswer` must reject requests with empty `Corpus` (or enforce explicit grounding instructions referencing only supplied chunks).
   - Prompts adapt friction to `TouchpointType`: `MORNING_CATALYST` (2-minute micro-drill) vs `EVENING_STREAK_SAVER` (15-second zero-friction recall card).
3. **Deterministic Output Validation (Go Guardrails)**:
   - Micro-drill prompt must be non-empty and bite-sized (**< 200 words**; evening streak saver even tighter, e.g., **<= 60 words**).
   - `EvaluateResponse.Score` strictly validated to `[0.0, 1.0]`, `Feedback` non-empty, `Passed` consistent with threshold (`score >= 0.6`).
   - Structured JSON output parsing with strict schema validation and bounded retry (max 1–2 retries on malformed/oversized output before returning a typed error).
4. **Offline Deterministic Tests**: Unit and contract tests run without live API keys using injected HTTP/client transport mocks and recorded fixtures.

## Gotchas & Constraints
- Batch `EmbedChunks` calls to respect Gemini embedding batch limits and preserve strict input-to-output ordering (`len(embeddings) == len(chunks)`).
- Never log raw API keys or user private corpus text at info level.
