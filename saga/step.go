package saga

// Step represents a single unit of work within a Saga transaction.
// Each step consists of an action (the main operation) and an optional
// compensation (the undo operation). Steps are executed in order, and if any
// step fails, the compensation of all successfully executed steps that have
// compensation enabled will be called in reverse order.
type Step struct {
	// Name of the step.
	Name string

	// Action is the main operation executed within a step's transaction.
	Action ActionFunc

	// Compensation - a compensating action that undoes the Action (if possible).
	// Called upon failure in subsequent steps.
	// Invoked when a subsequent step fails.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines (context that is passed through the action)
	//   - aroseErr: error from the previous action that needs compensation
	//
	// Note: Compensations should be idempotent as they might be called
	// multiple times in failure scenarios.
	Compensation CompensationFunc

	// CompensationOnFail needs to add the current compensation to the list of compensations.
	CompensationOnFail bool
}

// NewStep creates a new Step with the given name.
func NewStep(name string) Step {
	return Step{Name: name}
}

// WithAction sets the action function for the step and returns the modified step.
func (s Step) WithAction(fn ActionFunc) Step {
	s.Action = fn
	return s
}

// WithCompensation sets the compensation function for the step and returns the modified step.
func (s Step) WithCompensation(fn CompensationFunc) Step {
	s.Compensation = fn
	return s
}

// WithCompensationOnFail enables compensation for this step on failure.
func (s Step) WithCompensationOnFail() Step {
	s.CompensationOnFail = true
	return s
}
