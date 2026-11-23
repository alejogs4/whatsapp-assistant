Feature: WhatsApp Multi-Device Client Lifecycle and QR Pairing

  Scenario: Client rejects empty session DSN on construction
    Given a WhatsApp client config with an empty SessionDBPath
    When NewClient is called
    Then a configuration error is returned

  Scenario: Unpaired client renders QR code via injected writer and disconnects on context cancel
    Given a client lifecycle runner with a mock QR channel emitting a pairing code
    When Start is called and the context is subsequently cancelled
    Then the QR code is rendered to the configured output writer
    And Disconnect is invoked cleanly before Start returns
