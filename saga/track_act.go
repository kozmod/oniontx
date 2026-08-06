package saga

// ActType identifies an action applied to an execution track.
type ActType string

const (
	// ActCalled records an operation invocation.
	ActCalled ActType = "OperationCalled"
	// ActSucceeded records successful operation completion.
	ActSucceeded ActType = "OperationSucceeded"
	// ActFailed records operation failure and its optional cause.
	ActFailed ActType = "OperationFailed"
)

// Act describes a state transition applied to an execution track.
type Act struct {
	Type ActType
	Err  error
}

// NewTrackCalledAct creates an act indicating that an operation was called.
func NewTrackCalledAct() Act { return Act{Type: ActCalled} }

// NewTrackSucceededAct creates an act indicating successful operation completion.
func NewTrackSucceededAct() Act { return Act{Type: ActSucceeded} }

// NewTrackFailedAct creates an act indicating operation failure.
func NewTrackFailedAct(err error) Act { return Act{Type: ActFailed, Err: err} }
