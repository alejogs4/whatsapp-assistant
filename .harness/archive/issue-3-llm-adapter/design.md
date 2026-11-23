# Design: Issue #3 — Decoupled Gemini Adapter & Output Validation

## Decisions & Tradeoffs
1. **Official Go SDK (`google.golang.org/genai`)**: Replaces deprecated `generative-ai-go`. Encapsulated strictly inside `internal/adapters/llm/gemini/`.
2. **Structured JSON Output + Deterministic Go Guardrails**: Request `application/json` schema responses from Gemini, unmarshal into strict internal DTOs, then validate in pure Go (word count, non-empty prompt/feedback, score range `[0, 1]`, grounding check). Never trust raw LLM prose.
3. **Bounded Retry (Max 1 Retry)**: If the first response violates word limit or JSON schema, issue one corrective retry including the violation reason; if still invalid, return a typed error so callers can fail safely without infinite token burn.
4. **Batched Embeddings**: `EmbedChunks` splits large slices into safe batch sizes (e.g., 100 chunks) and preserves strict 1:1 index correspondence.

## Package & Contract Boundaries
- `internal/adapters/llm/gemini/`:
  - `config.go`: `Config{APIKey, TextModel, EmbeddingModel, MaxRetries, Timeout}`.
  - `prompts.go`: Pure functions building grounded system/user prompts for `MORNING_CATALYST`, `EVENING_STREAK_SAVER`, and `EvaluateAnswer`.
  - `validator.go`: Pure Go deterministic validators (`ValidateDrillRequest`, `ValidateDrillResponse`, `ValidateEvaluateRequest`, `ValidateEvaluateResponse`).
  - `client.go`: Concrete `genai.Client` wrapper + internal mockable transport interface.
  - `adapter.go`: `Adapter` implementing `touchpoint.LLMPort` and `learning.EmbeddingPort`.

## Verification Strategy
- **Tier 1 (`harness check`)**: `go vet ./...` and `go test -race ./...` (all tests strictly offline using fake transport & recorded JSON fixtures).
- **Tier 2 (BDD `.feature` specs)**: Verify grounding rejection, word-limit enforcement, score bounds, retry exhaustion, and embedding batch ordering.
