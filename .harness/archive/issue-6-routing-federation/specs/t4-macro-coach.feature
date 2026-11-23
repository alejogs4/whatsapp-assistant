Feature: Macro-Loop Milestone Synthesis and Prerequisite Recommendations (UC-4)

  Scenario: Path crossing mastery threshold triggers next-milestone recommendation
    Given User A's active path "RDD" has average concept StrengthScore >= 0.80
    When MacroCoach.EvaluateProgression is called
    Then SynthesisPort.SynthesizeMilestone is invoked with completed concepts and prerequisite context
    And a next-path proposal is returned without mutating existing paths automatically

  Scenario: Spontaneous curriculum question receives prerequisite-grounded guidance
    Given User A asks "What should I study after RDD?"
    When MacroCoach.AdviseNextSteps is called
    Then a structured roadmap recommendation is returned grounded in User A's completed and active paths
