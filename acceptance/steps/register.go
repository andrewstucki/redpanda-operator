package steps

import "github.com/cucumber/godog"

type stepDefinition struct {
	expression string
	step       interface{}
}

var steps []stepDefinition

func RegisterStep(expression string, step interface{}) {
	steps = append(steps, stepDefinition{expression, step})
}

func Scenarios(ctx *godog.ScenarioContext) {
	for _, step := range steps {
		ctx.Step(step.expression, step.step)
	}
}
