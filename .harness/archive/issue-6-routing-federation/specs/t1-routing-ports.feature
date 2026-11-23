Feature: Segregated Path Affinity and Synthesis Ports

  Scenario: Narrow routing ports keep domain packages free of external dependencies
    Given the PathAffinityPort and SynthesisPort interfaces in internal/touchpoint/ports.go
    When inspecting imports across internal/touchpoint
    Then zero external SDK imports exist
    And the Gemini adapter satisfies both ports with offline unit tests
