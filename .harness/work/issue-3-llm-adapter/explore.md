# Explore: Issue #3 — Decoupled Gemini Adapter & Output Validation

## Affected Topics
- `llm-adapter` (primary)
- `domain-core` (implements `touchpoint.LLMPort` and `learning.EmbeddingPort`)

## Current Codebase Reality
- `internal/touchpoint/ports.go` defines `LLMPort` (`GenerateMicroDrill`, `EvaluateAnswer`) with `DrillRequest`, `DrillResponse`, `EvaluateRequest`, `EvaluateResponse`.
- Issue #2 introduces `learning.EmbeddingPort` (`EmbedChunks`).
- No `internal/adapters/llm/` package or Gemini SDK dependency exists yet.

## Architectural Forks & Tradeoffs
1. **SDK Coupling vs Testability**:
   - Calling the Google GenAI SDK directly inside `GenerateMicroDrill` makes offline unit testing impossible without live credentials.
   - **Decision**: Wrap the concrete `google.golang.org/genai` client behind a minimal package-private `textGenerator` / `embedder` interface so unit and recorded contract tests run 100% offline under `go test -race ./...`.
2. **Validation & Retry Strategy**:
   - LLMs occasionally exceed word limits or emit malformed JSON.
   - **Decision**: Enforce deterministic Go validators (`< 200` words for `MORNING_CATALYST`, `<= 60` words for `EVENING_STREAK_SAVER`, non-empty grounded corpus, `0.0 <= score <= 1.0`, `Passed == (score >= 0.6)`), with at most 1 corrective retry before returning a typed validation error.
