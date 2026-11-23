Feature: Gemini SDK Configuration and Mockable Transport Wrapper

  Scenario: Missing API key fails fast during adapter construction
    Given a Gemini Config with an empty APIKey
    When NewAdapter is called
    Then a configuration error is returned without making any network calls

  Scenario: Adapter accepts an injected fake transport for offline tests
    Given a valid Config and a fake internal transport implementation
    When NewAdapterWithTransport is called
    Then the adapter initializes cleanly with default model names and retry limits
