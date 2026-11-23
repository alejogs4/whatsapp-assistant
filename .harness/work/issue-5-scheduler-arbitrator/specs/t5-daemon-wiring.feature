Feature: Daemon Composition Root Wiring and Graceful Shutdown

  Scenario: Daemon config validation fails fast on missing required environment settings
    Given an incomplete daemon configuration missing GEMINI_API_KEY
    When LoadConfig is called
    Then a descriptive configuration error is returned

  Scenario: Daemon wires dependencies cleanly and stops scheduler on context cancellation
    Given a valid daemon configuration with injectable adapters
    When the daemon starts and its context is cancelled
    Then the scheduler loop exits cleanly without goroutine leaks
