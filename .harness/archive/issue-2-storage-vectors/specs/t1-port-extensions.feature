Feature: Domain Port Extensions for Multi-Tenant Persistence and Embeddings

  Scenario: UserRepository supports phone lookup and active user listing
    Given the user feature ports in internal/user/ports.go
    When inspecting the UserRepository interface
    Then it declares FindByID, FindByPhoneNumber, FindAll, and Save
    And internal/user/domain remains free of external imports

  Scenario: TouchpointRepository enforces user_id scoping on session lookups
    Given the touchpoint feature ports in internal/touchpoint/ports.go
    When inspecting the TouchpointRepository interface
    Then FindByID requires both userID and sessionID
    And FindByUserAndDate queries sessions within a user's day window

  Scenario: Learning feature declares a provider-agnostic EmbeddingPort
    Given the learning feature ports in internal/learning/ports.go
    When inspecting EmbeddingPort
    Then it declares EmbedChunks accepting context and text chunks and returning float32 vectors
