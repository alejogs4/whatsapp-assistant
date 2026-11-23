# Design: Issue #6 — Hierarchical Path Router & Scatter-Gather Federation

## Decisions & Tradeoffs
1. **Strict 4-Stage Hierarchical Routing Pipeline (`InboundRouter`)**:
   - **Stage 0 (Tenant Resolution)**: Look up `User` via `UserRepository.FindByPhoneNumber(ctx, msg.SenderPhone)`. Unknown numbers are safely ignored (or receive a neutral onboarding notice without creating tenant data).
   - **Stage 1 (Operational Command Escape)**: Explicit slash commands (`/pause`, `/resume`, `/skip`, `/stats`, `/schedule`) execute deterministically via `CommandController` without calling the LLM or mutating `MasteryLedger`.
   - **Stage 2 (Drill State Guard)**: Check `TouchpointRepository.FindActiveSession(ctx, user.ID)` (expiring stale sessions first via `CheckExpiration`). If `AWAITING_ANSWER`, route directly to `PedagogicalEvaluator.EvaluateActiveSession` (UC-2).
   - **Stage 3 (Path Affinity & Intent Routing)**:
     - Invoke `PathAffinityClassifier` against the user's active `LearningPath`s.
     - **1 matched path** → Single-Path Grounded Q&A (retrieve chunks from that path's bag + answer grounded).
     - **>=2 matched paths** → `CrossPathSynthesizer` (Scatter-Gather concurrent `RetrieveChunks` across matched paths → unified synthesis).
     - **0 matched paths** → Intent fallback: Curriculum/progression intent → `MacroCoach` (UC-4); otherwise → General Assistant fallback.
2. **Bounded Context Read-Only Invariant**:
   - Scatter-Gather fans out concurrent `SourceCorpusPort.RetrieveChunks` calls using goroutines + `sync.WaitGroup` (or `errgroup`) with context cancellation, collects per-path results into isolated buckets, and **never** calls `MasteryLedgerRepository.Save`.
3. **Macro-Loop Milestone Synthesis (`MacroCoach`)**:
   - Computes path mastery ratio across ledger concepts; when a path meets target mastery threshold (e.g. >= 0.80 average strength) or user asks what to study next, queries prerequisite suggestions and proposes the next roadmap milestone via `SynthesisPort.SynthesizeMilestone`.

## Package & Contract Boundaries
- `internal/touchpoint/ports.go`: Add `IntentKind`, `AffinityResult`, `PathAffinityPort`, and `SynthesisPort` (`AnswerGrounded`, `SynthesizeCrossPath`, `SynthesizeMilestone`, `AnswerGeneral`).
- `internal/touchpoint/services/`:
  - `commands.go`: `CommandController`.
  - `scatter_gather.go`: `CrossPathSynthesizer`.
  - `macro_coach.go`: `MacroCoach`.
  - `router.go`: `InboundRouter` implementing `touchpoint.InboundMessageHandler`.

## Verification Strategy
- **Tier 1 (`harness check`)**: `go vet ./...` and `go test -race ./...` verifying concurrent Scatter-Gather race safety, routing precedence, command escapes, and zero ledger cross-contamination.
- **Tier 2 (BDD `.feature` specs)**: Verify every branch of the hierarchical router, Scatter-Gather concurrency & partial failure tolerance, and Macro-Loop milestone proposals.
