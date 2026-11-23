Feature: Outbound MessagingPort Implementation

  Scenario: SendText dispatches a WhatsApp conversation message to the normalized JID
    Given a MessagingPort adapter backed by a fake waSender
    When SendText is called with "+15551234567" and "Great answer!"
    Then the fake sender receives JID "15551234567@s.whatsapp.net" and body "Great answer!"

  Scenario: SendDrillPrompt validates non-empty prompt and formats for mobile readability
    Given a MessagingPort adapter backed by a fake waSender
    When SendDrillPrompt is called with an empty prompt
    Then a validation error is returned and no message is sent
    And when called with a valid drill prompt it sends a formatted message cleanly
