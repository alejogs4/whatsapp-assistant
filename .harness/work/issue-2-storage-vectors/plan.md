# Plan: Issue #2 — SQLite Multi-Tenant Persistence & Embedded Vector Store

## Affected Topics
- `storage-vectors`
- `domain-core`

## Outcomes (WHAT)
1. **Multi-Tenant Relational Persistence**: All domain aggregates (`User`, `LearningPath`, `MasteryLedger`, `TouchpointSession`) persist to and reload from SQLite with zero data loss and strict `user_id` isolation.
2. **Idempotent Schema Management**: Opening a fresh or existing SQLite database applies schema migrations cleanly with foreign keys and WAL mode enabled.
3. **Embedded Local Vector Retrieval**: Document chunks and float32 vector embeddings persist per `(user_id, path_id)` and support deterministic local k-NN cosine similarity search via `SourceCorpusPort`.
4. **Decoupled Port Contracts**: Repository and embedding interfaces satisfy upcoming Issues #3–#6 without leaking SQL or driver types into `internal/{user,learning,touchpoint}/domain/`.

## Constraints & Invariants
- Zero cross-user data leakage across every repository and vector query.
- Domain packages remain 100% standard-library Go.
- All timestamps persist in UTC RFC3339Nano without losing sub-second or timezone semantics.
- All integration tests execute deterministically against in-memory SQLite under `go test -race ./...`.

## Non-Goals
- Google Drive HTTP/OAuth ingestion (deferred to Drive adapter; `IngestPathCorpus` accepts an injected document fetcher or returns a clear unconfigured error when no fetcher is wired).
- PostgreSQL adapter or external managed vector databases.
- Live Gemini API calls during storage tests.
