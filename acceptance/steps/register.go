package steps

import "github.com/redpanda-data/redpanda-operator/acceptance/framework"

func init() {
	framework.RegisterStep(`^I add a user CRD for "([^"]*)" with an ACL$`, iAddAUserCRDWithAnACL)
	framework.RegisterStep(`^I create a user CRD for "([^"]*)" with authentication$`, iCreateAUserCRDWithAuthentication)
	framework.RegisterStep(`^I should be able to exercise that ACL$`, iShouldBeAbleToExerciseThatACL)
	framework.RegisterStep(`^"([^"]*)" should now exist in redpanda$`, shouldNowExistInRedpanda)
	framework.RegisterStep(`^there is a pre-existing user "([^"]*)"$`, thereIsAPreexistingUser)
	framework.RegisterStep(`^there is no user "([^"]*)"$`, thereIsNoUser)
}
