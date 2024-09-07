Feature: stub

  @aks @gke @eks @kind
  Scenario Outline:
    Given there is a stub
    When a user updates the stub key "<key>" to "<value>"
    Then the stub should have "<key>" equal "<value>"

    Examples:
        | key       | value |
        | stub      | true  |

  @aks @gke @eks @kind
  Scenario:
    Given I am in a random namespace
    And there is a stub
    When a user updates the stub key "foo" to "bar"
    Then the stub should have "foo" equal "bar"
