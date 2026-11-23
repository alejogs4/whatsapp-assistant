Feature: SQLite Connection Bootstrap and Idempotent Multi-Tenant Migrations

  Scenario: Fresh in-memory database runs all migrations cleanly
    Given a fresh in-memory SQLite database
    When Open and Migrate are executed
    Then tables users, learning_paths, mastery_ledger, touchpoint_sessions, and learning_path_chunks exist
    And PRAGMA foreign_keys is enabled

  Scenario: Migrations are idempotent across repeated startups
    Given a SQLite database that has already been migrated
    When Migrate is executed a second time
    Then no error is returned and existing schema state is preserved
