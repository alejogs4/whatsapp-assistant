Feature: Attention Arbitrator Selecting The ONE Thing

  Scenario: Most overdue due concept across active paths is selected
    Given a user with two ACTIVE learning paths "RDD" and "Algorithms" and one PAUSED path "Compilers"
    And "RDD" has a concept overdue by 3 days while "Algorithms" has a concept overdue by 1 day
    When SelectTheOneThing is called
    Then the overdue concept from "RDD" is selected
    And paused path "Compilers" is ignored

  Scenario: Calendar telemetry failure degrades gracefully to circadian baseline
    Given a CalendarTelemetryPort that returns an error
    When SelectTheOneThing is called during morning hours
    Then no error is returned and bandwidth defaults cleanly to HIGH
