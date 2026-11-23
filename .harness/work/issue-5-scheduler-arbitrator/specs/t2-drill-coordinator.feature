Feature: DrillCoordinator End-to-End Touchpoint Dispatch (UC-1)

  Scenario: Morning catalyst retrieves grounded chunks, opens session, records budget, and sends drill
    Given a user with DailyTouchesUsed equal to 0 and grounded chunks available for concept "stereotypes"
    When TriggerTouchpoint is called for MORNING_CATALYST
    Then LLMPort.GenerateMicroDrill is called with the retrieved corpus chunks
    And a TouchpointSession is saved in state AWAITING_ANSWER
    And User.DailyTouchesUsed is incremented to 1 and saved
    And MessagingPort.SendDrillPrompt is invoked once

  Scenario: Exhausted daily budget blocks dispatch without calling LLM or WhatsApp
    Given a user with DailyTouchesUsed equal to 2 on the current local calendar day
    When TriggerTouchpoint is called
    Then ErrTouchpointBudgetExhausted is returned
    And neither LLMPort nor MessagingPort is called
