Feature: Micro-Drill Generation and Answer Evaluation with Bounded Retries

  Scenario: Oversized drill triggers one corrective retry and succeeds
    Given a fake transport whose first response has 250 words and second response has 80 words
    When GenerateMicroDrill is called with valid grounded corpus chunks
    Then the adapter retries once with feedback about the word limit
    And returns the valid 80-word DrillResponse

  Scenario: Persistent malformed JSON fails after max retries
    Given a fake transport that always returns malformed JSON
    When EvaluateAnswer is called
    Then the adapter stops after the configured retry limit and returns a typed error

  Scenario: Valid evaluation JSON parses into EvaluateResponse
    Given a fake transport returning valid JSON with score 0.85 and grounded feedback
    When EvaluateAnswer is called
    Then EvaluateResponse is returned with Score 0.85, Passed true, and non-empty Feedback
