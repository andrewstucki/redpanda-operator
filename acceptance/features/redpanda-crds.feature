Feature: Redpanda CRDs
  @skip:gke @skip:aks @skip:eks
  Scenario: Managing Redpanda clusters
    Given I apply Kubernetes manifest:
    """
    ---
    apiVersion: cluster.redpanda.com/v1alpha2
    kind: Redpanda
    metadata:
      name: my-cluster
    spec:
      clusterSpec:
        statefulset:
          replicas: 1
    """
    When cluster "my-cluster" is available
    Then I should be able to connect to cluster "my-cluster"