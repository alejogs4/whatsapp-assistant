Feature: 2-Touch Proactive Scheduler and Zero-Nag Silent Backoff (UC-1 & UC-3)

  Scenario: Morning catalyst fires once during user's local morning hour and is idempotent on repeat ticks
    Given a user in timezone "America/New_York" with MorningCatalystHour 8
    When RunTick is executed at 08:00 local time and again at 08:05 local time
    Then exactly 1 MORNING_CATALYST touchpoint is dispatched

  Scenario: Unanswered morning drill expires silently with zero follow-up nudges
    Given a user with an AWAITING_ANSWER morning session past its ExpiresAt timestamp
    When RunTick runs at 14:00 local time
    Then the session is persisted as EXPIRED
    And MessagingPort sends 0 messages

  Scenario: Evening streak saver adapts based on morning completion and stops at 2/2 budget
    Given a user whose morning session expired unanswered and EveningReflectionHour is 20
    When RunTick runs at 20:00 local time
    Then an EVENING_STREAK_SAVER recall card is dispatched and DailyTouchesUsed becomes 2
    And a subsequent tick at 20:15 sends 0 messages
