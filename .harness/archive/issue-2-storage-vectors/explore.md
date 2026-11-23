# Explore: Issue #2 — SQLite Multi-Tenant Persistence & Embedded Vector Store

## Affected Topics
- `storage-vectors` (primary)
- `domain-core` (minor port extensions)

## Current Codebase Reality
- Domain aggregates and ports exist in `internal/{shared,user,learning,touchpoint}/` with 28/28 unit tests passing (`go test -race ./...`).
- No `internal/adapters/storage/sqlite/` package or SQLite dependency in `go.mod` yet.
- Existing port gaps discovered:
  1. `user.UserRepository` only has `FindByID` and `Save`; Issues #4 and #5 need `FindByPhoneNumber` and `FindAll`.
  2. `touchpoint.TouchpointRepository.FindByID(ctx, id)` is missing `userID` scoping (violates strict multi-tenancy) and needs `FindByUserAndDate` for evening streak checks.
  3. `learning.SourceCorpusPort.RetrieveChunks(..., query string, limit int)` takes raw text, so the SQLite adapter needs an injected `EmbeddingPort` interface rather than importing Gemini directly.

## Architectural Forks & Tradeoffs
1. **SQLite Driver (`mattn/go-sqlite3` vs `modernc.org/sqlite`)**:
   - *Option A (`mattn/go-sqlite3`, CGo)*: Battle-tested with `whatsmeow` `sqlstore`, fast, standard. Requires CGo toolchain.
   - *Option B (`modernc.org/sqlite`, pure Go)*: Zero CGo, fast cross-compilation, also supported by `whatsmeow` via `sqlstore.New("sqlite", ...)`.
   - **Recommendation**: Use `modernc.org/sqlite` (or driver-agnostic `database/sql` with `sqlite` dialect) so `go test -race ./...` works reliably across macOS/Linux CI without CGo friction, while keeping SQL standard SQLite3.
2. **Vector Similarity (`BLOB` cosine in Go vs `sqlite-vec` C extension)**:
   - *Option A (float32 `BLOB` + pure-Go cosine k-NN)*: Zero native extension distribution headaches, trivially fast for per-user/per-path corpora (<50k chunks per path), 100% deterministic unit tests.
   - *Option B (`sqlite-vec`)*: Faster at >100k vectors per partition, but requires C extension loading and platform-specific binaries.
   - **Recommendation**: Option A (float32 `BLOB` + pure-Go cosine k-NN) behind `SourceCorpusPort`; swap to `sqlite-vec` later only if corpus scale demands it.

## Missing Deterministic Guardrails
- Need in-memory SQLite integration tests proving cross-tenant isolation (User A cannot read/overwrite User B's paths, ledgers, sessions, or vector chunks).
