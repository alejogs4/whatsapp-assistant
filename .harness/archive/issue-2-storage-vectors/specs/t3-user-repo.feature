Feature: SQLite UserRepository Persistence

  Scenario: Save and reload a User by ID and normalized phone number
    Given a migrated in-memory SQLite database
    And a User with timezone "America/New_York", phone "+15551234567", and 1 touchpoint used
    When the User is saved via UserRepository.Save
    Then FindByID returns an identical User including LastTouchSentAt and CircadianBaseline
    And FindByPhoneNumber returns the same User

  Scenario: Upsert updates daily touchpoint counter without duplicating rows
    Given an existing persisted User with DailyTouchesUsed equal to 1
    When RecordTouchpointSent is called and Save is executed again
    Then FindByID returns DailyTouchesUsed equal to 2
    And FindAll returns exactly 1 User
