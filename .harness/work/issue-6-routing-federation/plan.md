# Plan: Issue #6 — Hierarchical Path Router & Scatter-Gather Federation

## Affected Topics
- `routing-federation`
- `domain-core`

## Outcomes (WHAT)
1. **Drill State Isolation & Command Escapes**: Active drill answers route exclusively to `PedagogicalEvaluator`, while explicit operational commands (`/pause`, `/resume`, `/skip`, `/stats`, `/schedule`) work anytime without corrupting `MasteryLedger`.
2. **Path Affinity Single-Path Grounded Q&A**: Spontaneous questions matching one active `LearningPath` load only that path's grounded corpus bag and answer without bleeding other paths.
3. **Concurrent Scatter-Gather Cross-Path Federation**: Inquiries spanning ≥2 active paths fan out concurrent corpus retrievals and synthesize a unified multi-disciplinary answer while keeping every `MasteryLedger` untouched.
4. **Macro-Loop Mentorship (UC-4)**: Detects path milestone mastery thresholds and answers curriculum-progression inquiries with prerequisite-aware next-path recommendations.

## Non-Goals
- Automatic Google Drive folder creation for newly accepted paths.
- Multi-turn conversational memory beyond the active session/query context.
- Automated WhatsApp group chat routing.
