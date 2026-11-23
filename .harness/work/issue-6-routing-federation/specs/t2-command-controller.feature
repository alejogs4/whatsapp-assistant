Feature: Operational Command Controller

  Scenario: /schedule updates circadian baseline without invoking the LLM
    Given a persisted User with MorningCatalystHour 8
    When the user sends "/schedule morning 9"
    Then User.CircadianBaseline.MorningCatalystHour is updated to 9 and saved
    And a confirmation reply is sent via MessagingPort with zero LLM calls

  Scenario: /skip expires an active session without mutating MasteryLedger
    Given a user with an AWAITING_ANSWER TouchpointSession
    When the user sends "/skip"
    Then the session is marked EXPIRED
    And MasteryLedgerRepository.Save is never called
