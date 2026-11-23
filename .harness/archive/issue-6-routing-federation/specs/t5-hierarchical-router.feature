Feature: Hierarchical Inbound Message Router

  Scenario: Active drill answer routes to PedagogicalEvaluator before Path Affinity
    Given User A has an active AWAITING_ANSWER TouchpointSession
    When User A sends "Information Holder stores state"
    Then PedagogicalEvaluator is invoked and PathAffinityPort is NOT called

  Scenario: Spontaneous single-path query routes to Grounded Q&A for that path only
    Given User A has no active TouchpointSession
    And PathAffinityPort matches only path "RDD"
    When User A sends "Remind me what a Coordinator stereotype does"
    Then SourceCorpusPort.RetrieveChunks is called only for "RDD"
    And the grounded answer is sent via MessagingPort

  Scenario: Spontaneous cross-path query triggers Scatter-Gather federation
    Given User A has no active TouchpointSession
    And PathAffinityPort matches both "Algorithms" and "AI"
    When User A sends "How do graph traversals relate to neural attention?"
    Then CrossPathSynthesizer is invoked and the unified answer is sent via MessagingPort

  Scenario: Unbound curriculum query routes to MacroCoach
    Given User A has no active TouchpointSession
    And PathAffinityPort matches 0 paths with intent CURRICULUM
    When User A sends "What should I learn next?"
    Then MacroCoach is invoked and the recommendation is sent via MessagingPort
