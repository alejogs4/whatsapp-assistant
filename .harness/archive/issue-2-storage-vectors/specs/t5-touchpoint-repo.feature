Feature: SQLite TouchpointRepository Persistence

  Scenario: Active session lookup returns only AWAITING_ANSWER sessions for the owning user
    Given User A has one COMPLETED session and one AWAITING_ANSWER session
    And User B has an AWAITING_ANSWER session
    When FindActiveSession is called for User A
    Then only User A's AWAITING_ANSWER session is returned

  Scenario: Cross-tenant FindByID is rejected and daily audit returns day-scoped sessions
    Given User A has a morning session created today and an older session from yesterday
    When FindByID is called with User B's ID and User A's session ID
    Then a not-found error is returned
    And FindByUserAndDate for User A covering today returns only today's session
