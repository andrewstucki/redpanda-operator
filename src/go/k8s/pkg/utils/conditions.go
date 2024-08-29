package utils

import (
	"errors"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"
)

type conditionErrorType struct {
	err    error
	reason string
}

type ConditionFn func(reason string, err error) metav1.Condition

type ConditionErrorHandler struct {
	types    []conditionErrorType
	fallback string
	handler  ConditionFn
}

func NewConditionErrorHandler(handler ConditionFn, fallback string) *ConditionErrorHandler {
	return &ConditionErrorHandler{
		handler:  handler,
		fallback: fallback,
	}
}

func (c *ConditionErrorHandler) Register(target error, reason string) *ConditionErrorHandler {
	c.types = append(c.types, conditionErrorType{target, reason})
	return c
}

func (c *ConditionErrorHandler) Handle(err error) (metav1.Condition, error) {
	for _, t := range c.types {
		if errors.Is(err, t.err) {
			return c.handler(t.reason, err), nil
		}
	}

	return c.handler(c.fallback, err), err
}

// StatusConditionConfigs takes a set of existing conditions and conditions of a
// desired state and merges them idempotently to create a series of ConditionApplyConfiguration
// values that can be used in a server-side apply.
func StatusConditionConfigs(existing []metav1.Condition, generation int64, conditions []metav1.Condition) []*metav1ac.ConditionApplyConfiguration {
	now := metav1.Now()
	configurations := []*metav1ac.ConditionApplyConfiguration{}

	for _, condition := range conditions {
		existingCondition := apimeta.FindStatusCondition(existing, condition.Type)
		if existingCondition == nil {
			configurations = append(configurations, conditionToConfig(generation, now, condition))
			continue
		}

		if existingCondition.Status != condition.Status {
			configurations = append(configurations, conditionToConfig(generation, now, condition))
			continue
		}

		if existingCondition.Reason != condition.Reason {
			configurations = append(configurations, conditionToConfig(generation, now, condition))
			continue
		}

		if existingCondition.Message != condition.Message {
			configurations = append(configurations, conditionToConfig(generation, now, condition))
			continue
		}

		if existingCondition.ObservedGeneration != generation {
			configurations = append(configurations, conditionToConfig(generation, now, condition))
			continue
		}

		configurations = append(configurations, conditionToConfig(generation, existingCondition.LastTransitionTime, *existingCondition))
	}

	return configurations
}

func conditionToConfig(generation int64, now metav1.Time, condition metav1.Condition) *metav1ac.ConditionApplyConfiguration {
	return metav1ac.Condition().
		WithType(condition.Type).
		WithStatus(condition.Status).
		WithReason(condition.Reason).
		WithObservedGeneration(generation).
		WithLastTransitionTime(now).
		WithMessage(condition.Message)
}
