# Implementation Tasks & GitHub Issues Breakdown

This technical breakdown splits the proposal in [`ai_specs/proposal.md`](file:///Users/alejandrogarciaserna/Documents/github/whatsapp-assistant/ai_specs/proposal.md) into incremental, decoupled work units tracked in [`alejogs4/whatsapp-assistant`](https://github.com/alejogs4/whatsapp-assistant).

---

### [Issue #1: `feat(core): implement domain aggregates and hexagonal ports`](https://github.com/alejogs4/whatsapp-assistant/issues/1)
* **Description**: Establish pure Go business invariants with zero external dependencies.
* **Deliverables**:
  * `internal/core/domain/user.go`: User entity, circadian baselines, daily budget state.
  * `internal/core/domain/learning_path.go`: Path aggregate, concept graph references.
  * `internal/core/domain/mastery_ledger.go`: Spaced repetition state, concept scores, misconceptions.
  * `internal/core/domain/touchpoint_session.go`: State machine (`PENDING` $\rightarrow$ `AWAITING_ANSWER` $\rightarrow$ `COMPLETED`).
  * `internal/core/ports/`: Interfaces for `MessagingPort`, `SourceCorpusPort`, `CalendarTelemetryPort`, `LLMPort`, `RepositoryPort`.
* **Testing**: Comprehensive domain unit tests with 100% invariant coverage.

---

### [Issue #2: `feat(storage): implement sqlite multi-tenant persistence and embedded vector store`](https://github.com/alejogs4/whatsapp-assistant/issues/2)
* **Description**: Create SQLite storage adapter with strict `user_id` partitioning and local vector similarity.
* **Deliverables**:
  * `internal/adapters/storage/sqlite/`: Migrations for `users`, `learning_paths`, `mastery_ledger`, `touchpoint_audit`, and `chunks`.
  * Local k-NN cosine similarity search against chunk vector embeddings.
  * Implement `RepositoryPort` and `SourceCorpusPort`.
* **Testing**: In-memory SQLite integration tests.

---

### [Issue #3: `feat(llm): implement decoupled gemini adapter and output validation`](https://github.com/alejogs4/whatsapp-assistant/issues/3)
* **Description**: Implement `ports.LLMPort` using Google Gemini SDK with strict schema guards.
* **Deliverables**:
  * `internal/adapters/llm/gemini/`: Implements `EmbedChunks`, `GenerateMicroDrill`, `EvaluateAnswer`.
  * Deterministic validation: Bite-sized prompt length (<200 words), zero hallucination invariants.
* **Testing**: Mock-based unit tests and recorded contract tests.

---

### [Issue #4: `feat(whatsapp): implement whatsmeow multi-device transport adapter`](https://github.com/alejogs4/whatsapp-assistant/issues/4)
* **Description**: Create native WhatsApp Web multi-device connection implementing `ports.MessagingPort`.
* **Deliverables**:
  * `internal/adapters/whatsapp/`: Client initialization, terminal QR code login, session SQLite persistence.
  * Inbound message event hook dispatching to core router.
  * Outbound text and interactive prompt delivery.

---

### [Issue #5: `feat(scheduler): implement 2-touch proactive daemon and attention arbitrator`](https://github.com/alejogs4/whatsapp-assistant/issues/5)
* **Description**: Build background execution runner enforcing the 2-touch daily notification budget.
* **Deliverables**:
  * `internal/daemon/scheduler.go`: Cron trigger for 8:00 AM Morning Catalyst and 8:00 PM Streak Saver.
  * `internal/core/services/arbitrator.go`: Zero-decision-fatigue logic designating **The ONE Thing**.
  * Zero-nag backoff policy: Silent backoff when drills are left on read.

---

### [Issue #6: `feat(routing): implement hierarchical path router and scatter-gather federation`](https://github.com/alejogs4/whatsapp-assistant/issues/6)
* **Description**: Route inbound messages across bounded context bags and coordinate multi-path queries.
* **Deliverables**:
  * `internal/core/services/router.go`: Active drill guard $\rightarrow$ Path Affinity Classifier $\rightarrow$ Fallback intent router.
  * Scatter-Gather multi-path query synthesis (Algorithms $\cap$ AI).
  * Macro-loop milestone synthesis and prerequisite recommendation engine.
