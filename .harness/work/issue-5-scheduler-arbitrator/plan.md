# Plan: Issue #5 — 2-Touch Proactive Daemon & Attention Arbitrator

## Affected Topics
- `scheduler-arbitrator`
- `domain-core`

## Outcomes (WHAT)
1. **Zero-Decision-Fatigue Arbitration**: `AttentionArbitrator` selects a single highest-leverage `(LearningPath, ConceptKey)` across active paths adapted to real-time `CognitiveBandwidth`.
2. **End-to-End Micro-Drill Loop (UC-1, UC-2, UC-3)**:
   - Morning Catalyst dispatches a grounded 2-minute micro-drill and opens an `AWAITING_ANSWER` session.
   - Inbound drill answers are evaluated via `PedagogicalEvaluator`, updating `MasteryLedger` SM-2 state and completing the session.
   - Evening Reflection adapts based on morning completion (celebration vs 15-second streak-saver recall card).
3. **Strict 2-Touch Budget & Zero-Nag Backoff**: Unanswered sessions expire silently with zero follow-up nudges; repeated ticks within the same circadian hour never send duplicates.
4. **Runnable Composition Root**: `cmd/daemon` boots SQLite, Gemini, WhatsApp, fallback Calendar telemetry, and the proactive Scheduler with graceful shutdown.

## Non-Goals
- Full Google Calendar OAuth integration (interface supported; fallback baseline adapter ships in MVP).
- Multi-path spontaneous routing and Scatter-Gather federation (Issue #6).
