Feature: SQLite LearningPath and MasteryLedger Multi-Tenant Persistence

  Scenario: LearningPath queries enforce strict user_id isolation
    Given User A owns LearningPath "RDD" and User B owns LearningPath "Algorithms"
    When FindByUser is called for User A
    Then only "RDD" is returned
    And FindByID for User A with User B's path ID returns a not-found error

  Scenario: MasteryLedger persists SM-2 state, misconceptions, and due reviews
    Given a MasteryLedger for User A on path "RDD" and concept "stereotypes" with two misconceptions
    And NextReviewAt set in the past
    When Save is called and FindDueReviews is queried for User A before now
    Then the concept is returned with identical StrengthScore, EaseFactor, IntervalDays, and Misconceptions
    And FindDueReviews for User B returns an empty slice
