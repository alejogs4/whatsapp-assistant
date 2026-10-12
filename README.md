# WhatsApp AI Adaptive Learning Assistant

> **An autonomous, energy-adaptive AI learning coach on WhatsApp that eliminates cognitive friction by dynamically matching study tasks to your real-time mental bandwidth.**

[![Go Version](https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Architecture](https://img.shields.io/badge/architecture-Hexagonal%20%7C%20Screaming-orange)](docs/architecture.md)
[![Domain Purity](https://img.shields.io/badge/domain-Pure%20Go%20(0%20deps)-brightgreen)](AGENTS.md)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

---

## 🎯 The Problem & Job-to-be-Done (JTBD)

High-ambition engineers and knowledge workers rarely fail to learn due to a lack of ambition or calendar time. They fail because of **activation friction and cognitive exhaustion** after demanding workdays.

Traditional learning tools (flashcard decks, video courses, rigid study planners) assume infinite, uniform willpower every day. When you open a 50-page chapter after an 8-hour sprint of complex debugging, cognitive inertia wins $\rightarrow$ guilt triggers $\rightarrow$ the learning habit drops to zero.

### The Job-to-be-Done
> *"When my mental energy fluctuates throughout the day, dynamically match learning friction to my cognitive state—delivering deep exercises when I am fresh and zero-friction 15-second active recall cards when I am drained—so my habit momentum never breaks."*

---

## ⚡ Core Product Pillars

### 1. The Strict 2-Touch Daily Budget & Zero-Nag Backoff
To protect your attention, the assistant is hard-capped at **2 proactive interactions per day**:

* 🌅 **8:00 AM — Morning Catalyst (The ONE Thing)**:  
  Evaluates active curriculums and designates *The ONE Most Important Concept* to eliminate choice paradox. Delivers a focused, 2-minute in-chat micro-drill to jumpstart the day.
* 🌙 **8:00 PM — Evening Streak Saver (Low-Friction Recall)**:  
  Serves a 15-second active recall prompt designed to consolidate knowledge before sleep without demanding willpower.
* 🛡️ **Zero-Nag Backoff Policy**:  
  If a prompt is left on read or ignored, the assistant **backs off silently**. It never spams, pesters, or creates guilt notifications.

### 2. Grounded Bounded Contexts (Learning Paths)
The assistant does not produce generic trivia from the open web. It operates over isolated **Learning Paths** grounded in your primary source materials:
* **Grounded RAG (NotebookLM Model)**: Ingests reference books, PDFs, and notes from linked Google Drive folders into a local embedded vector store.
* **Mastery Ledger & Misconception Tracking**: Tracks retention using an adaptive SM-2 spaced repetition engine, isolating concept mastery and identifying specific cognitive misconceptions per path.
* **Strict Context Isolation**: Learning paths behave as isolated bounded contexts. Mastering *Software Design (RDD)* never pollutes or desynchronizes your mastery in *Distributed Systems*.

### 3. Dual-Loop Coaching & Attention Arbitration
* **Micro-Loop (Daily)**: In-chat drill delivery, active response evaluation, instant Socratic feedback, and misconception logging.
* **Macro-Loop (Milestones)**: As mastery matures, the coach negotiates roadmap milestones (*"You've solidified RDD fundamentals; should we branch into Design Patterns or Go Concurrency?"*).
* **Cross-Path Scatter-Gather Federation**: Spontaneous learner queries that span multiple paths (e.g., *"How do graph algorithms apply to AI neural networks?"*) are federated across paths without requiring the user to navigate separate menus.

---

## 🏗️ Architectural Foundations

Built with **Hexagonal Architecture (Ports & Adapters)** and **Package-by-Feature (Screaming Architecture)** in Go:

```
cmd/
  daemon/                  # Main entrypoint: boots SQLite, adapters, wires features, runs whatsmeow
internal/
  shared/                  # Shared Kernel: typed domain IDs (UserID, PathID, SessionID), sentinel errors
  user/                    # Identity & Circadian Baselines
    domain/                # User aggregate root, 2-touch daily budget, timezone rollover
    ports.go               # UserRepository interface
  learning/                # Curriculum & Knowledge Context
    domain/                # LearningPath & MasteryLedger aggregates, SM-2 spaced repetition
    ports.go               # LearningPathRepository, MasteryLedgerRepository, SourceCorpusPort
  touchpoint/              # Adaptive Proactive Drills
    domain/                # TouchpointSession aggregate, state machine, silent backoff rules
    ports.go               # TouchpointRepository, MessagingPort, LLMPort, CalendarTelemetryPort
  adapters/
    whatsapp/              # whatsmeow multi-device client adapter
    gdrive/                # Google Drive corpus ingestor
    gcalendar/             # Google Calendar cognitive load reader
    llm/                   # Gemini adapter implementing ports.LLMPort
    storage/sqlite/        # SQLite repositories & local vector storage
```

### Key Engineering Invariants
1. **Pure Domain (`internal/*/domain/`)**: 100% standard-library Go with **zero external dependencies** (no ORMs, no external UUID libraries, no AI SDKs in domain packages).
2. **Aggregate Decoupling by Identity**: Aggregates reference each other strictly by typed IDs (`UserID`, `PathID`, etc.) from `internal/shared/`, preventing Go `import cycle not allowed` compilation errors.
3. **Multi-Tenant from Day 1**: Every domain aggregate and database table is strictly partitioned by `user_id`.
4. **Hexagonal Ports**: Concrete infrastructure (SQLite, Gemini, whatsmeow) lives strictly behind clean Go interfaces in `ports.go`.

---

## 🛠️ Technology Stack

| Layer | Technology | Rationale |
|:---|:---|:---|
| **Language** | **Go (Golang 1.25)** | Single compiled binary, lean memory footprint (~30MB), native concurrency. |
| **Messaging** | **whatsmeow** | Native Go WhatsApp Web multi-device connection via QR pairing (no third-party SaaS required). |
| **Intelligence** | **Google Gemini** | High-speed structured outputs and embeddings behind decoupled `ports.LLMPort`. |
| **Persistence & Vectors** | **SQLite (embedded k-NN)** | Self-contained, portable storage with local cosine vector similarity search. |
| **Telemetry & Sources** | **Google Calendar & Drive** | Calendar meeting density infers mental bandwidth; Drive ingests grounded textbooks/notes. |

---

## 🚀 Getting Started

### Prerequisites
* [Go 1.25+](https://golang.org/dl/)
* WhatsApp mobile app (for scanning the pairing QR code during adapter boot)

### Clone & Test
```bash
# Clone the repository
git clone https://github.com/alejogs4/whatsapp-assistant.git
cd whatsapp-assistant

# Verify domain invariants and test suite
go test -v ./...

# Run static analysis
go vet ./...
```

---

## 🗺️ Roadmap & Milestones

The project is built incrementally following Spec-Driven Development (SDD):

- [x] **[Phase 1: Pure Feature Domains & Hexagonal Ports](https://github.com/alejogs4/whatsapp-assistant/issues/1)** (`#1`)  
  Pure Go aggregates (`User`, `LearningPath`, `MasteryLedger`, `TouchpointSession`), state machine transitions, and explicit port contracts.
- [ ] **[Phase 2: SQLite Persistence & Embedded Vector Store](https://github.com/alejogs4/whatsapp-assistant/issues/2)** (`#2`)  
  Multi-tenant schema, local chunk embeddings, and k-NN cosine similarity search.
- [ ] **[Phase 3: Decoupled LLM Adapter (Gemini)](https://github.com/alejogs4/whatsapp-assistant/issues/3)** (`#3`)  
  Bite-sized micro-drill generation, active recall answer evaluation, and embedding creation.
- [ ] **[Phase 4: WhatsApp Transport Adapter (`whatsmeow`)](https://github.com/alejogs4/whatsapp-assistant/issues/4)** (`#4`)  
  Multi-device terminal QR pairing and inbound message routing.
- [ ] **[Phase 5: 2-Touch Proactive Daemon & Attention Arbitrator](https://github.com/alejogs4/whatsapp-assistant/issues/5)** (`#5`)  
  Circadian scheduler, "The ONE Thing" arbitration, and zero-nag backoff policy.
- [ ] **[Phase 6: Multi-Path Router & Cross-Domain Scatter-Gather](https://github.com/alejogs4/whatsapp-assistant/issues/6)** (`#6`)  
  Path affinity classifier and federated cross-path query synthesis.

---

## 📖 Authoritative Documentation

* [Product Vision & JTBD Framework](docs/product.md)
* [Technical Architecture & Hexagonal Boundaries](docs/architecture.md)
* [Domain Model & RDD Object Interactions](docs/domain-model.md)
* [Agent Engineering Rules & Architectural Invariants](AGENTS.md)
