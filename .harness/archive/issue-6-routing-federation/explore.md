# Explore: Issue #6 — Hierarchical Path Router & Scatter-Gather Federation

## Affected Topics
- `routing-federation` (primary)
- `domain-core` (extends intelligence ports, preserves Bounded Context isolation)
- `scheduler-arbitrator` (delegates active drill answers to `PedagogicalEvaluator`)

## Current Codebase Reality
- `docs/domain-model.md` sections 6 & 7 define the exact Hierarchical Classifier Pipeline and Scatter-Gather Cross-Path Federation.
- `touchpoint.LLMPort` currently only defines `GenerateMicroDrill` and `EvaluateAnswer`.
- Issue #4 introduces `touchpoint.InboundMessage` and `touchpoint.InboundMessageHandler`.
- Issue #5 introduces `PedagogicalEvaluator`.

## Architectural Forks & Tradeoffs
1. **Single Monolithic `LLMPort` vs Segregated Intelligence Ports**:
   - *Option A (Interface Segregation)*: Keep `touchpoint.LLMPort` focused on micro-drills and add narrow `IntentClassifierPort` / `SynthesisPort` interfaces (all implementable by `adapters/llm/gemini`).
   - **Recommendation**: Option A — avoids bloating mocks in Issue #5 tests while keeping Gemini as a single concrete adapter implementing all narrow ports.
2. **Command Escape During Active Drills**:
   - What if a user has an active `AWAITING_ANSWER` drill and sends `/pause` or `/skip`?
   - **Recommendation**: Check explicit slash commands (`/pause`, `/resume`, `/skip`, `/stats`, `/schedule`) *before* routing to `PedagogicalEvaluator` so a user can always escape or pause without corrupting their `MasteryLedger` score.
