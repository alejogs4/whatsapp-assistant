Feature: Phone/JID Normalization and 1:1 DM Event Filtering

  Scenario: E.164 phone numbers convert cleanly to and from WhatsApp JIDs
    Given a phone number "+1 (555) 123-4567"
    When PhoneToJID is called
    Then the resulting JID has User "15551234567" and Server "s.whatsapp.net"
    And JIDToPhone returns "+15551234567"

  Scenario: Self messages, group messages, and status broadcasts are ignored
    Given synthetic WhatsApp events for a self-sent message, a group chat message, and a status broadcast
    When ExtractInboundMessage is called for each event
    Then all three return ok=false

  Scenario: Standard 1:1 text and extended-text DMs extract cleanly
    Given a 1:1 DM event containing an ExtendedTextMessage "Encapsulation hides state"
    When ExtractInboundMessage is called
    Then ok=true is returned with normalized E.164 SenderPhone and exact Text
