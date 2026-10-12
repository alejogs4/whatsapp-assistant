# WhatsApp AI Assistant — Technical Architecture & System Design

## Architecture Philosophy: Hexagonal (Ports & Adapters)
To ensure long-term stability and prevent vendor lock-in, the system strictly separates core learning and coaching logic from external I/O protocols (WhatsApp, Google APIs, LLMs, Databases).

```
                      +-----------------------------------+
                      |         DRIVING ADAPTERS          |
                      |                                   |
                      |  [Scheduler / Cron Daemon]        |
                      |  [whatsmeow Inbound Listener]     |
                      +-----------------+-----------------+
                                        | (Calls Inbound Ports)
                                        v
+-----------------------------------------------------------------------------------+
| CORE DOMAIN (`internal/core`)                                                     |
|                                                                                   |
|  [Domain Services]                                                                |
|    - AttentionArbitrator (Picks The ONE Thing)                                    |
|    - EnergyEvaluator (Correlates load & telemetry)                                |
|    - PedagogicalEngine (Refines path skills & recall)                             |
|                                                                                   |
|  [Domain Aggregates]                                                              |
|    - User (Circadian profile, notification budget state)                          |
|    - LearningPath (Curriculum state, topic graph)                                 |
|    - MasteryLedger (Concept scores, misconceptions, spaced repetition)            |
|                                                                                   |
|  [Outbound Ports / Interfaces]                                                    |
|    - MessagingPort (SendText, SendDrill)                                          |
|    - SourceCorpusPort (RetrieveChunks, IngestDocuments)                           |
|    - CalendarTelemetryPort (GetDailyLoad)                                         |
|    - LLMPort (GenerateDrill, SynthesizeMilestone)                                 |
|    - RepositoryPort (UserRepo, PathRepo, LedgerRepo)                              |
+-----------------------------------------------------------------------------------+
                                        | (Implemented by Outbound Adapters)
                                        v
                      +-----------------+-----------------+
                      |         DRIVEN ADAPTERS           |
                      |                                   |
                      |  `adapters/whatsapp/whatsmeow`    |
                      |  `adapters/drive/gdrive`          |
                      |  `adapters/calendar/gcalendar`    |
                      |  `adapters/llm/gemini_openai`     |
                      |  `adapters/storage/sqlite`        |
                      +-----------------------------------+
```

## Directory Structure (Screaming Architecture & Package by Feature)

The codebase organizes business capabilities into product/feature packages rather than generic technical layers. Each feature package encapsulates its domain invariants and contracts. To eliminate cross-aggregate memory coupling and satisfy Go's strict import cycle constraints, aggregates reference each other strictly by **Identity (ID)** via the shared kernel.

```
cmd/
  daemon/                  # Main entrypoint: initializes DB, adapters, wires features, boots whatsmeow
internal/
  shared/                  # Shared Kernel: value types, domain IDs (UserID, PathID, SessionID), tenant context
  user/                    # Identity & Circadian Baselines
    domain/                # User aggregate root, daily 2-touch budget, circadian baselines
    ports.go               # UserRepository interface
  learning/                # Curriculum & Knowledge Context
    domain/                # LearningPath & MasteryLedger aggregates, spaced repetition state
    ports.go               # LearningPathRepository, MasteryLedgerRepository, SourceCorpusPort
  touchpoint/              # Adaptive Proactive Drills
    domain/                # TouchpointSession aggregate, session state machine, backoff rules
    ports.go               # TouchpointRepository, MessagingPort, LLMPort, CalendarTelemetryPort
    services/              # Arbitrator ("The ONE Thing"), DrillRunner, ProgressCoach
  adapters/
    whatsapp/              # whatsmeow multi-device client adapter
    gdrive/                # Google Drive source documents fetcher
    gcalendar/             # Google Calendar load reader
    llm/                   # LLM client (prompt formatting, structured outputs)
    storage/sqlite/        # SQLite repositories with strict user_id scoping
```

### Aggregate Decoupling & Multi-Tenancy Invariant
Aggregates across features reference each other strictly by **Identity (`UserID`, `PathID`)**, NEVER by direct struct pointers or object graphs. This guarantees:
1. **Compilation Safety**: Completely eliminates Go `import cycle not allowed` compilation errors.
2. **True Bounded Contexts**: `touchpoint` cannot directly mutate `user` or `learning` aggregates in memory.
3. **Multi-Tenant Partitioning**: Aligns with SQLite row-level scoping (`user_id` and `path_id`).


## Persistence & Multi-Tenancy Boundary
- Every table/collection is strictly partitioned by `user_id`:
  - `users`: ID, phone number, timezone, circadian baseline.
  - `learning_paths`: ID, user_id, title, status (active/paused/completed), source_folder_id.
  - `mastery_ledger`: ID, user_id, path_id, concept_key, strength_score, next_review_at.
  - `touchpoint_audit`: ID, user_id, date, morning_sent, evening_sent, responses.
- SQLite is used for MVP (portable single file, backed by CGo required by `whatsmeow`), with standard repository interfaces allowing seamless transition to PostgreSQL for production multi-tenant deployments.

## Embedded Vector Retrieval Engine (RAG)
- **Storage Strategy**: SQLite stores document chunks, metadata, and raw vector embeddings (`BLOB` or `sqlite-vec` extension).
- **Partitioning**: Strictly scoped to `user_id` and `path_id` (`learning_path_chunks`).
- **Retrieval Pipeline**:
  1. Ingest PDF / Markdown files from Google Drive folder linked to the Learning Path.
  2. Semantic chunking with token overlap.
  3. Generate embeddings via `LLMPort.EmbedChunks(ctx, chunks)`.
  4. Perform local k-NN cosine similarity search directly against SQLite for question generation and concept grounding.
  5. Zero external network calls or separate database clusters required.

## LLM Port Abstraction & Provider Decoupling
To honor the **Dependency Inversion Principle**, the domain knows zero vendor SDKs. It depends strictly on `ports.LLMPort`:
```go
type LLMPort interface {
    EmbedChunks(ctx context.Context, chunks []string) ([][]float32, error)
    GenerateMicroDrill(ctx context.Context, req DrillRequest) (*Drill, error)
    EvaluateAnswer(ctx context.Context, req AnswerEvaluationRequest) (*EvaluationResult, error)
    SynthesizeMilestone(ctx context.Context, req MilestoneRequest) (*MilestoneRecommendation, error)
}
```
- **Initial Implementation**: `adapters/llm/gemini` (using Google GenAI SDK for Go).
- **Swappability**: Alternative adapters (`adapters/llm/openai`, `adapters/llm/anthropic`, `adapters/llm/ollama`) can be substituted via environment config (`LLM_PROVIDER=gemini`) with zero changes to domain logic.
