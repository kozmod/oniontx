package saga

// Step represents a single unit of work within a local saga.
// Each step consists of an action and an optional compensation operation.
// Steps are executed in order. If any step fails, compensations for completed
// steps are called in reverse order.
type Step struct {
	// name of the step.
	name string

	// action is the main operation executed for the step.
	action Operation

	// compensation is an optional operation that attempts to undo the action.
	// Compensation should be idempotent because recovery code may be retried by
	// callers after a failed saga execution.
	compensation Operation

	// compensationRequired adds this step to the compensation list before its
	// action runs, so the step can compensate itself if its own action fails.
	compensationRequired bool
}

// NewStep creates a new Step with the given name.
func NewStep(name string) Step {
	return Step{name: name}
}

// WithAction sets the action operation for the step.
func (s Step) WithAction(op Operation) Step {
	s.action = op
	return s
}

// WithCompensation sets the compensation operation for the step.
func (s Step) WithCompensation(op Operation) Step {
	s.compensation = op
	return s
}

// WithCompensationRequired enables compensation for this step even if its own
// action fails. Without this flag, only successfully completed steps are
// considered for compensation.
func (s Step) WithCompensationRequired() Step {
	s.compensationRequired = true
	return s
}
