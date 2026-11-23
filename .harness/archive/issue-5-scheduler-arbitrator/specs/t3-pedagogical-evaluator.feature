Feature: PedagogicalEvaluator Inbound Drill Evaluation (UC-2)

  Scenario: Valid answer transitions session to COMPLETED, updates MasteryLedger, and sends feedback
    Given an active TouchpointSession in state AWAITING_ANSWER for concept "stereotypes"
    And LLMPort.EvaluateAnswer returns Score 0.9, Passed true, and grounded Feedback
    When EvaluateActiveSession is called with the user's answer
    Then the session transitions to COMPLETED and is persisted
    And MasteryLedger.RecordReview is called with score 0.9 and saved
    And MessagingPort.SendText delivers the feedback to the user

  Scenario: Expired session is not evaluated and does not mutate MasteryLedger
    Given a TouchpointSession whose ExpiresAt is in the past
    When EvaluateActiveSession is called
    Then ErrSessionExpired is returned, the session is saved as EXPIRED, and MasteryLedger is unchanged
