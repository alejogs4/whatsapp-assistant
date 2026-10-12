# WhatsApp AI Assistant — Product Characterization

## Vision
An adaptive AI learning coach operating over WhatsApp that eliminates activation friction by dynamically matching study tasks to the user's real-time cognitive energy, continuously refining pedagogical strategies across grounded learning paths.

## Product Definition Framework
A product is not a list of features or capabilities. It defines:
- **Target User & Context**: Who is using it and in what emotional/operational state.
- **Job-to-be-Done (JTBD)**: The real-world progress or burden being solved (the "why hire this").
- **Core Loop & Triggers**: Trigger -> Action -> Value -> Feedback.
- **Guardrails & Failure Modes**: Where the product fails or annoys the user (noise, hallucinations, interruptions).

## Pillars Under Definition
1. **Target Persona & Context**: High-ambition professional striving for continuous mastery, constrained not by absolute time, but by **variable mental energy and cognitive fatigue**.
2. **Core Problem & Job-to-be-Done (JTBD)**:
   - *Problem*: High activation resistance caused by a mismatch between current cognitive energy and the friction of the proposed learning activity.
   - *JTBD Statement*: *"When my cognitive energy fluctuates throughout the day, dynamically match learning friction to my energy state (deep practice when fresh, low-friction maintenance like flashcards/audio when depleted) so I never break momentum."*
3. **Adaptive Modality Spectrum**:
   - *High Energy*: Deliberate practice, problem solving, rigorous reading/exercises.
   - *Low Energy*: Spaced-repetition flashcards (Anki-style), high-signal video/audio, micro-reviews.
4. **Energy Detection & Forecasting Model**:
   - *Decouple Availability vs Energy*: Latency/silence reflects physical availability (meetings, family), NOT mental energy.
   - *Calibrated Task Telemetry*: Track completion and drop-off rates against the *predefined cognitive load* of each activity (e.g., deep exercises vs micro-flashcards).
   - *Intake Baseline*: Initial circadian profile to seed priors before telemetry accumulates.
5. **Proactivity Boundaries & Notification Budget**:
   - *Budget*: Strictly capped at 2 proactive touches/day.
   - *Touchpoint 1 (Morning Catalyst)*: Proactive kickoff framing a single high-priority commitment.
   - *Touchpoint 2 (Evening Reflection / Micro-Drill)*: Check-in paired with zero-friction in-chat review (e.g. quick active-recall prompt).
   - *Silence / Non-Response Rule*: Strict zero-nag policy. If left on read or ignored, the agent backs off completely until the next scheduled boundary.
6. **Self-Learning Scope & Guardrails (Learning Paths)**:
   - *Encapsulated Unit*: Discrete **Learning Paths** (e.g., *Software Design via RDD*).
   - *Path Composition*:
     1. **Source Corpus**: Grounded reference materials (book chapters, notes, diagrams — NotebookLM model).
     2. **Pedagogical Skill**: Learns the user's specific comprehension bottlenecks, effective analogies, and preferred challenge formats.
     3. **Mastery Ledger**: Spaced repetition state, concept comprehension score, and active misconceptions.
   - *Architectural Guardrail*: The assistant never alters global application runtime code; self-improvement is strictly confined to evolving path-level pedagogical strategies and drill artifacts.
7. **Integration Model & Permissions**:
   - **Google Drive** *(Content Ingestion)*: Read access to specified folders containing reference books, notes, and PDFs to feed Learning Path corpora.
   - **Google Calendar** *(Cognitive Load Telemetry)*: Read-only access to assess daily meeting density and focus blocks to predict cognitive fatigue.
   - **Email**: *EXCLUDED* (Violates YAGNI, high privacy risk, zero value for core learning loop).
8. **Multi-User Model & Privacy Boundary**:
   - *Phased Rollout*: Single-user MVP for rapid personal feedback, but strictly designed for multiple concurrent users from day one.
   - *Data Isolation*: Every user has completely segregated learning paths, mastery ledgers, and source corpora—zero cross-user data leakage.
   - *User Experience*: Seamless onboarding per user without administrative overhead.
9. **Curriculum Progression & Macro-Mentorship**:
   - *Dual-Loop Coaching Model*:
     - **Micro-Loop (Daily)**: Energy-adapted drills, active recall, accountability nudges within active paths.
     - **Macro-Loop (Milestones)**: Cross-path milestone synthesis, prerequisite graph navigation, and roadmap proposals.
   - *Concurrent Multi-Path Support*: User can maintain multiple concurrent paths (e.g., Software Design, AI, Algorithms). Paths may be interrelated or orthogonal.
   - *Daily Attention Arbitration (Zero Decision Fatigue)*: The morning kickoff autonomously designates **The ONE Most Important Thing** across active paths to eliminate choice paradox and protect momentum. The user wakes up to a single clear target, not a menu of chores.
   - *Collaborative Steering (Human-in-the-Loop)*:
     - The coach suggests next steps grounded in foundational dependencies (e.g. *Algorithms $\rightarrow$ Compilers*).
     - The user steers or redirects according to personal ambition (e.g. choosing *Networks* instead).
     - **Macro-Skill Polishing**: Every steering choice directly refines the coach's understanding of the user's technical appetite, career direction, and learning velocity.
