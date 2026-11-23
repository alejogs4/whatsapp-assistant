# Topic: `routing-federation`

## Purpose
Hierarchical inbound message router (`internal/touchpoint/services/router.go`), Path Affinity Classifier, **Scatter-Gather** cross-path query federation, operational command controller, and **Macro-Loop** curriculum milestone mentorship.

## Scope & Related Issue
- **Issue #6**: `feat(routing): implement hierarchical path router and scatter-gather federation`
- **Implements**:
  1. **Drill State Guard (Tier 1 Priority)**: Checks `TouchpointRepository.FindActiveSession(ctx, userID)` (expiring stale sessions first). If an active `AWAITING_ANSWER` session exists and message is not an explicit system escape command (e.g. `/pause`, `/skip`), routes to `PedagogicalEvaluator` (UC-2).
  2. **Operational Command Controller**: Intercepts explicit utility commands (`/pause`, `/resume`, `/stats`, `/schedule HH:MM`) without touching `MasteryLedger` or LLM tokens.
  3. **Path Affinity Classifier**: Classifies spontaneous messages against the user's active `LearningPath` titles/concepts/embeddings → resolves 0, 1, or N matching Context Bags.
  4. **Single-Path Grounded Q&A**: Loads chunks + ledger context for the single matched path and answers grounded in its corpus.
  5. **Scatter-Gather Cross-Path Federation**: When affinity spans ≥2 active paths (e.g., *Algorithms ∩ AI*), fans out concurrent `RetrieveChunks` calls across matched paths (`errgroup` / goroutines with context cancellation) and synthesizes a unified multi-disciplinary answer.
  6. **Macro-Loop Milestone Coach (UC-4)**: Handles curriculum progression queries (*"What should I study after RDD?"*) and detects when a path crosses target mastery threshold to propose next roadmap milestones.

## Architectural Invariants
1. **Active Drill Isolation**: Casual/command messages must not corrupt `MasteryLedger` spaced-repetition scores; active drill answers always route to `PedagogicalEvaluator`.
2. **Bounded Context Integrity**: Scatter-Gather reads chunks from Path A and Path B in parallel, but **NEVER** mutates or cross-contaminates either path's `MasteryLedger`.
3. **Strict Multi-Tenancy**: Router resolves `User` from normalized sender phone number and passes `userID` through every repository, corpus, and synthesis call.
4. **Concurrency Safety**: Scatter-Gather fan-out must be race-free (`go test -race`), respect `ctx` timeouts, and degrade gracefully if one path returns zero chunks.

## Gotchas & Constraints
- Unbound general queries (0 matching paths, non-command, non-curriculum) route to a lightweight general assistant fallback without mutating any path state.
- Keep LLM classification/synthesis behind narrow port interfaces so the entire routing tree and scatter-gather concurrency are deterministically unit-tested with fakes.
