Feature: Inbound WhatsApp Event Dispatching

  Scenario: Valid 1:1 DM is forwarded to InboundMessageHandler
    Given a dispatcher wired to a mock InboundMessageHandler
    When a valid 1:1 DM event arrives
    Then HandleInbound is invoked once with the normalized InboundMessage

  Scenario: Filtered events and handler errors do not crash the listener
    Given a dispatcher whose InboundMessageHandler returns an error
    When a group message arrives followed by a valid 1:1 DM
    Then the group message does not invoke the handler
    And the handler error on the 1:1 DM is logged without panicking
