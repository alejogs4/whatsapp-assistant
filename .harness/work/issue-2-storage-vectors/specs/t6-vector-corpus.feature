Feature: Embedded SQLite Vector Store and Local Cosine k-NN Search

  Scenario: RetrieveChunks ranks chunks by cosine similarity within the target path
    Given a fake EmbeddingPort returning deterministic unit vectors
    And 3 chunks stored under User A and Path "RDD"
    When RetrieveChunks is called with a query vector closest to chunk 2 and limit 2
    Then chunk 2 is ranked first followed by the second closest chunk
    And each returned GroundedChunk has Similarity in [0, 1]

  Scenario: Vector search strictly isolates users and learning paths
    Given User A has chunks in Path "RDD" and Path "Algorithms"
    And User B has an identical chunk in Path "RDD"
    When RetrieveChunks is called for User A on Path "RDD"
    Then no chunks from Path "Algorithms" or User B are returned

  Scenario: Zero-norm embedding vectors do not produce NaN similarity
    Given a stored chunk with a zero-norm vector
    When RetrieveChunks runs cosine similarity
    Then the similarity score is 0.0 and no panic or NaN occurs
