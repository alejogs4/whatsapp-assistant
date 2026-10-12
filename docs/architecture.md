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

## Directory Structure (Screaming Architecture)
```
cmd/
  daemon/                  # Main entrypoint: initializes DB, adapters, wires domain, boots whatsmeow
internal/
  core/
    domain/                # Pure Go domain structs & business invariants (Zero external deps)
      user.go              # User entity, circadian baselines, daily budget state
      learning_path.go     # LearningPath aggregate, prerequisites, source refs
      mastery_ledger.go    # Spaced repetition items, concept retention ledger
      modality.go          # High vs low energy activity definitions
    ports/                 # Contracts defining inputs and outputs
      inbound.go           # CoachService, InboundMessagePort
      messaging.go         # MessagingPort
      corpus.go            # SourceCorpusPort
      telemetry.go         # CalendarTelemetryPort
      llm.go               # LLMPort
      repository.go        # Storage interfaces
    services/              # Use cases orchestrating domain rules
      arbitrator.go        # Designates "The ONE Thing" for morning touchpoint
      drill_runner.go      # Executes in-chat active-recall micro-drills
      progress_coach.go    # Macro-loop milestone evaluation & path recommendations
  adapters/
    whatsapp/              # whatsmeow multi-device client adapter
    gdrive/                # Google Drive source documents fetcher
    gcalendar/             # Google Calendar load reader
    llm/                   # LLM client (prompt formatting, structured outputs)
    storage/sqlite/        # SQLite repositories with strict user_id scoping
```

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
