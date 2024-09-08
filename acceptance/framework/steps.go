package framework

import "github.com/cucumber/godog"

type stepDefinition struct {
	expression string
	step       interface{}
}

var registeredSteps []stepDefinition

func RegisterStep(expression string, step interface{}) {
	registeredSteps = append(registeredSteps, stepDefinition{expression, step})
}

func getSteps(ctx *godog.ScenarioContext) {
	for _, step := range registeredSteps {
		ctx.Step(step.expression, step.step)
	}
}
