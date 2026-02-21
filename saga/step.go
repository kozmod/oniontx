package saga

// Step of the [Saga].
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
	Compensation CompensationFunc

	// CompensationOnFail needs to add the current compensation to the list of compensations.
	CompensationOnFail bool
}

func NewStep(name string) Step {
	return Step{Name: name}
}

func (s Step) WithAction(fn ActionFunc) Step {
	s.Action = fn
	return s
}

func (s Step) WithCompensation(fn CompensationFunc) Step {
	s.Compensation = fn
	return s
}

func (s Step) WithCompensationOnFail() Step {
	s.CompensationOnFail = true
	return s
}
