@isolated @cluster:basic
Feature: User CRDs
  Scenario: Managing Users
    Given there is no user "bob"
    When I create a user CRD for "bob" with authentication
    Then "bob" should now exist in redpanda

  @skip:gke
  Scenario: Managing User ACLs
    Given there is a pre-existing user "bob"
    When I add a user CRD for "bob" with an ACL
    Then I should be able to exercise that ACL
