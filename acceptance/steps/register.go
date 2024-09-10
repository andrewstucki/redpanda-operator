package steps

import "github.com/redpanda-data/redpanda-operator/acceptance/framework"

func init() {
	framework.RegisterStep(`^cluster "([^"]*)" is available$`, checkClusterAvailability)
	framework.RegisterStep(`^there is no user "([^"]*)" in cluster "([^"]*)"$`, thereIsNoUser)
	framework.RegisterStep(`^I create CRD-based users for cluster "([^"]*)":$`, iCreateCRDbasedUsers)
	framework.RegisterStep(`^"([^"]*)" should exist and be able to authenticate to the "([^"]*)" cluster$`, shouldExistAndBeAbleToAuthenticateToTheCluster)
}
