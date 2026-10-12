# AGENTS.md — Engineering & Agent Instructions

This document defines the architectural principles, domain constraints, and technical conventions governing the WhatsApp AI Adaptive Learning Assistant. All AI coding agents and contributors must adhere to these directives.

---

## 1. Authoritative References

Before proposing or modifying code, consult the primary design documents:

* **Product Vision & JTBD**: [docs/product.md](file:///Users/alejandrogarciaserna/Documents/github/whatsapp-assistant/docs/product.md)  
  Defines the persona, core Job-to-be-Done (energy-adaptive friction matching), the strict 2-touch daily budget, and non-goals.
* **Technical Architecture**: [docs/architecture.md](file:///Users/alejandrogarciaserna/Documents/github/whatsapp-assistant/docs/architecture.md)  
  Specifies the Hexagonal (Ports & Adapters) layout, screaming directory structure, SQLite embedded vector search, and LLM port abstractions.
* **Domain Model & RDD**: [docs/domain-model.md](file:///Users/alejandrogarciaserna/Documents/github/whatsapp-assistant/docs/domain-model.md)  
  Details Responsibility-Driven Design object stereotypes, collaboration sequences (UC-1 to UC-4), Hierarchical Intent Routing, and Scatter-Gather cross-path federation.

---

## 2. Technology Stack Breakdown

* **Language**: Go (Golang) — Single compiled binary, lean memory footprint, native concurrency.
* **Architecture Pattern**: **Hexagonal Architecture (Ports & Adapters)**.
* **Storage & Vectors**: SQLite with embedded vector similarity (cosine k-NN on embeddings), strictly partitioned by `user_id`.
* **Inbound Messaging Driver**: `whatsmeow` (multi-device WhatsApp protocol in Go).
* **AI / Intelligence**: Google Gemini (primary `adapters/llm` implementation behind `ports.LLMPort`).
* **Telemetry & Sources**: Google Calendar (cognitive load assessment) and Google Drive (source corpus ingestion).

---

## 3. Core Architectural Invariants

### Invariant 1: Pure Core Domain (`internal/core/domain/`)
* **ZERO external dependencies**: The core domain must contain standard-library Go only.
* Never import framework code, database drivers, WhatsApp libraries (`whatsmeow`), or AI SDKs inside `internal/core/domain`.
* Business rules, aggregates ([`User`](file:///Users/alejandrogarciaserna/Documents/github/whatsapp-assistant/docs/domain-model.md#L132), [`LearningPath`](file:///Users/alejandrogarciaserna/Documents/github/whatsapp-assistant/docs/domain-model.md#L141), [`MasteryLedger`](file:///Users/alejandrogarciaserna/Documents/github/whatsapp-assistant/docs/domain-model.md#L150), [`TouchpointSession`](file:///Users/alejandrogarciaserna/Documents/github/whatsapp-assistant/docs/domain-model.md#L158)), and state transitions live here.

### Invariant 2: Explicit Ports (`internal/core/ports/`)
* Communication between the domain and outside infrastructure occurs strictly across interfaces defined in `ports`.
* **Driving Adapters** (cron daemon, whatsmeow listener) call inbound ports.
* **Driven Adapters** (`adapters/storage`, `adapters/llm`, `adapters/whatsapp`) implement outbound ports (`LLMPort`, `MessagingPort`, `RepositoryPort`).

### Invariant 3: Strict Multi-Tenancy Boundary
* Every table, vector index, and domain aggregate is strictly scoped to `user_id`.
* Zero cross-user data leakage. Learning paths and mastery ledgers belong solely to the owning user.

### Invariant 4: Proactivity Budget & Zero-Nag Backoff
* The system is capped at **2 proactive touches per day** (Morning Catalyst and Evening Reflection).
* If a touchpoint is left unanswered, enforce a silent backoff. Never spam or nag the user.

---

## 4. Directory Structure (Screaming Architecture)

```
cmd/
  daemon/                  # Main daemon: boots config, SQLite, adapters, domain wiring
internal/
  core/
    domain/                # Pure Go domain entities, aggregates, business invariants
    ports/                 # Inbound and outbound interface definitions
    services/              # Use-case orchestrators (arbitrator, drill runner, coach)
  adapters/
    whatsapp/              # whatsmeow client adapter
    gdrive/                # Google Drive corpus ingestor
    gcalendar/             # Google Calendar telemetry reader
    llm/                   # Gemini/OpenAI adapter implementing ports.LLMPort
    storage/sqlite/        # SQLite repositories & vector storage
```

---

## 5. Working Rules for Coding Agents

1. **Test-First (TDD)**: Write failing domain unit tests before implementing business logic in `internal/core/domain`.
2. **No Leaky Abstractions**: Keep storage queries, SQL strings, and API payloads isolated inside `internal/adapters`.
3. **Respect Bounded Contexts**: A `LearningPath` is an isolated context bag. Do not bleed one path's mastery state into another during cross-domain queries.
