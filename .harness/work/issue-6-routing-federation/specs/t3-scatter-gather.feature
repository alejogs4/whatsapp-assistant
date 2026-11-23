Feature: Scatter-Gather Cross-Path Query Federation

  Scenario: Multi-path query fans out concurrently and synthesizes without cross-contaminating ledgers
    Given User A has active paths "Algorithms" and "AI" with distinct grounded chunks
    When CrossPathSynthesizer.Synthesize is called for both paths
    Then SourceCorpusPort.RetrieveChunks is invoked concurrently for "Algorithms" and "AI" scoped to User A
    And SynthesisPort.SynthesizeCrossPath receives both isolated chunk sets
    And MasteryLedgerRepository.Save is called 0 times

  Scenario: Partial empty retrieval on one path still produces a valid synthesized answer
    Given "Algorithms" returns 3 chunks and "AI" returns 0 chunks
    When CrossPathSynthesizer.Synthesize is called
    Then synthesis succeeds using available chunks without error or race condition
