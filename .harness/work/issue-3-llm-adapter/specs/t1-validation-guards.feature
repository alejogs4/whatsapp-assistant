Feature: Grounded Prompt Builders and Deterministic Output Validators

  Scenario: Empty corpus is rejected before calling the LLM
    Given a DrillRequest with an empty Corpus slice
    When ValidateDrillRequest is called
    Then a grounding validation error is returned

  Scenario: Bite-sized word limits are enforced per TouchpointType
    Given a DrillResponse with 215 words for MORNING_CATALYST
    When ValidateDrillResponse is called
    Then a prompt-too-long validation error is returned
    And a DrillResponse with 45 words for EVENING_STREAK_SAVER passes validation

  Scenario: Evaluation score bounds and Passed consistency are enforced
    Given an EvaluateResponse with Score 1.25
    When ValidateEvaluateResponse is called
    Then an invalid-score validation error is returned
    And an EvaluateResponse with Score 0.75 and non-empty Feedback passes with Passed true
