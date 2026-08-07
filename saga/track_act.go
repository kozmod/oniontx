package saga

type trackActType string

const (
	trackActCalled    trackActType = "Called"
	trackActSucceeded trackActType = "Succeeded"
	trackActFailed    trackActType = "Failed"
)

type trackAct struct {
	typeID trackActType
	err    error
}

func newTrackCalledAct() trackAct { return trackAct{typeID: trackActCalled} }

func newTrackSucceededAct() trackAct { return trackAct{typeID: trackActSucceeded} }

func newTrackFailedAct(err error) trackAct { return trackAct{typeID: trackActFailed, err: err} }
