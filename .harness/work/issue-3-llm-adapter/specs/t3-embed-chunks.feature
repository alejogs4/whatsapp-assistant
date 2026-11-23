Feature: Batched Chunk Embeddings via Gemini Adapter

  Scenario: EmbedChunks preserves order across multiple batches
    Given 125 text chunks and a fake transport with batch size limit 50
    When EmbedChunks is called
    Then the transport is invoked 3 times
    And 125 embedding vectors are returned in exact input order

  Scenario: Empty chunk slice returns empty embeddings without transport calls
    Given an empty chunk slice
    When EmbedChunks is called
    Then an empty slice and nil error are returned with zero transport calls
