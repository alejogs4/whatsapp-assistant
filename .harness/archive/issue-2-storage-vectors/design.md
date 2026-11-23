# Design: Issue #2 — SQLite Multi-Tenant Persistence & Embedded Vector Store

## Decisions & Tradeoffs
1. **Pure-Go SQLite (`modernc.org/sqlite`) via `database/sql`**: Chosen over CGo `go-sqlite3` to keep builds portable and `-race` tests frictionless while remaining compatible with `whatsmeow` `sqlstore`. Connection configured with `PRAGMA journal_mode=WAL`, `PRAGMA foreign_keys=ON`, and `PRAGMA busy_timeout=5000`.
2. **Float32 Little-Endian `BLOB` + Pure-Go Cosine k-NN**: Chosen over `sqlite-vec` to avoid C extension loading complexity. For each `(user_id, path_id)` partition, chunks are scanned and ranked by cosine similarity with deterministic tie-breaking by `chunk_id`. Zero-norm vectors return similarity `0` (never `NaN`).
3. **Narrow `EmbeddingPort` in `internal/learning/ports.go`**: `EmbedChunks(ctx context.Context, chunks []string) ([][]float32, error)` decouples `SourceCorpusPort` from Gemini (Issue #3) and allows deterministic fake embedders in storage tests.
4. **Strict `user_id` Partitioning Everywhere**: Every table stores `user_id` and every `SELECT`/`UPDATE`/`DELETE` predicates on `user_id`. `TouchpointRepository.FindByID` is updated to accept `userID shared.UserID`.

## Package & Contract Boundaries
- `internal/user/ports.go`: Add `FindByPhoneNumber(ctx, phone string)` and `FindAll(ctx) ([]*domain.User, error)`.
- `internal/learning/ports.go`: Add `EmbeddingPort` (`EmbedChunks`) and `StoreChunks(ctx, userID, pathID, chunks []string)` (or chunk ingestion helper) on `SourceCorpusPort`.
- `internal/touchpoint/ports.go`: Update `FindByID(ctx, userID shared.UserID, id shared.SessionID)` and add `FindByUserAndDate(ctx, userID shared.UserID, dayStart, dayEnd time.Time) ([]*domain.TouchpointSession, error)`.
- `internal/adapters/storage/sqlite/`:
  - `db.go` + `migrations.go`: `Open(dsn string) (*DB, error)` and idempotent `Migrate(ctx) error`.
  - `user_repo.go`: `UserRepository` implementation.
  - `learning_path_repo.go`, `mastery_ledger_repo.go`: Learning repositories.
  - `touchpoint_repo.go`: `TouchpointRepository` implementation.
  - `corpus_store.go` + `vector.go`: Float32 `BLOB` codec, cosine similarity, and `SourceCorpusPort` implementation.

## Verification Strategy
- **Tier 1 (`harness check`)**: `go vet ./...` and `go test -race ./...` covering all repositories and vector search against `:memory:` SQLite.
- **Tier 2 (BDD `.feature` specs)**: Scenario-by-scenario verification of multi-tenant isolation, idempotent migrations, SM-2 ledger round-trips, session expiration queries, and k-NN ranking accuracy.
