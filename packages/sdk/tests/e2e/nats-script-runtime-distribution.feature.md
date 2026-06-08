Feature: NATS script runtime distribution

  Scenario: Built package runs the substrate and record-store contract end to end
    Given the package has a clean final-form distribution build
    And the distribution exposes ESM and CommonJS entrypoints
    When a consumer imports the ESM entrypoint from dist
    And starts embedded JetStream NATS through RuntimeSubstrate
    And writes two versions of a script record through ScriptRecordStore
    Then the consumer can read the exact first and second KV revisions
    And a deleted script record returns RecordDeletedOrStale from the distribution
    And the embedded NATS server stops cleanly
