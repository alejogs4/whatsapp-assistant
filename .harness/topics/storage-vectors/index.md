# Topic: `storage-vectors`

## Purpose
SQLite persistence adapter (`internal/adapters/storage/sqlite/`) providing strict multi-tenant relational storage and embedded local vector similarity search (RAG) with zero external database infrastructure.

## Scope & Related Issue
- **Issue #2**: `feat(storage): implement sqlite multi-tenant persistence and embedded vector store`
- **Implements**:
  - `user.UserRepository`
  - `learning.LearningPathRepository`
  - `learning.MasteryLedgerRepository`
  - `learning.SourceCorpusPort` (chunk storage & k-NN cosine retrieval; Drive ingestion delegated or plugged in)
  - `touchpoint.TouchpointRepository`

## Architectural Invariants
1. **Strict `user_id` Partitioning**: Every table (`users`, `learning_paths`, `mastery_ledger`, `touchpoint_sessions`, `learning_path_chunks`) includes `user_id` and every query filters by `user_id` (composite indexes on `(user_id, ...)`). Zero cross-tenant reads or writes.
2. **No Leaky Abstractions**: All SQL strings, schema migrations, row scanning, and vector encoding live strictly inside `internal/adapters/storage/sqlite/`.
3. **Embedded Local Vector k-NN**: Embeddings stored as compact float32 `BLOB`s in `learning_path_chunks` scoped by `(user_id, path_id)`. Local cosine similarity search in Go (with deterministic tie-breaking and unit-norm / zero-vector safety) — no external managed vector DB.
4. **Single SQLite Driver Strategy**: Align driver choice with `whatsmeow` (Issue #4) and enable WAL mode + `PRAGMA foreign_keys = ON` + busy timeout.
5. **Deterministic Testing**: All repository and vector search tests run against in-memory SQLite (`:memory:`) with `-race` enabled under `go test -race ./...`.

## Key Tables
- `users`: `id`, `phone_number` (unique), `timezone`, `morning_catalyst_hour`, `evening_reflection_hour`, `daily_touches_used`, `last_touch_sent_at`.
- `learning_paths`: `id`, `user_id`, `title`, `source_folder_id`, `status`.
- `mastery_ledger`: `(user_id, path_id, concept_key)` PK, `strength_score`, `interval_days`, `ease_factor`, `next_review_at`, `misconceptions_json`.
- `touchpoint_sessions`: `id`, `user_id`, `path_id`, `concept_key`, `state`, `touchpoint_type`, `prompt_text`, `user_answer`, `evaluation_feedback`, `created_at`, `expires_at`, `completed_at`.
- `learning_path_chunks`: `id`, `user_id`, `path_id`, `content`, `embedding_blob`, `source_ref`, `created_at`.

## Gotchas & Constraints
- **Driver**: Uses pure-Go `modernc.org/sqlite` (`database/sql` driver `"sqlite"`). For `:memory:` DSNs, `Open()` enforces `SetMaxOpenConns(1)` so all queries share the single in-memory database; file-backed DBs enable `PRAGMA journal_mode = WAL;`.
- `SourceCorpusPort.RetrieveChunks` accepts `query string`; `CorpusStore` injects `learning.EmbeddingPort` and optional `FolderFetcher` so storage stays decoupled from Gemini and Google Drive.
- Always parse/store timestamps in RFC3339Nano UTC to avoid timezone drift across `User.Timezone` calculations.
- Status: **Implemented & verified** (`internal/adapters/storage/sqlite/`, commit `d425134`).
