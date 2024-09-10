# @cluster:sasl
Feature: User CRDs
#   Background: Cluster available
    # Given cluster "sasl" is available

  Scenario: Managing Users
    Given there is no user "bob" in cluster "sasl"
    When I create CRD-based users for cluster "sasl":
      | name | password | mechanism     | acls |
      | bob  |          | SCRAM-SHA-256 |      |
    Then "bob" should exist and be able to authenticate to the "sasl" cluster