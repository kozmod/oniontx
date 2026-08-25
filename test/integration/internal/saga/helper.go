package saga

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kozmod/oniontx/saga"
)

func printResult(t *testing.T, res saga.Result, err error) {
	t.Helper()

	var sb strings.Builder
	writef := func(format string, a ...any) {
		_, _ = sb.WriteString(fmt.Sprintf(format, a...))
	}

	writef("[result]:\n")
	writef("%v\n", res)
	writef("[execution error]:\n%v\n", err)
	writef("\n[steps]:")

	for _, step := range res.Steps {
		writef("\nstep [%d#%s]:\n", step.StepPosition, step.StepName)

		switch {
		case len(step.Action.Errors) > 0:
			writef("  action errors (%d):\n", len(step.Action.Errors))
			for i, e := range step.Action.Errors {
				writef("    %d: %v\n", i, e)
			}
		default:
			writef("  action errors: none\n")
		}

		switch {
		case len(step.Compensation.Errors) > 0:
			writef("  compensation errors (%d):\n", len(step.Compensation.Errors))
			for i, e := range step.Compensation.Errors {
				writef("    %d: %v\n", i, e)
			}
		default:
			writef("  compensation errors: none\n")
		}

		writef("  -----\n")
		writef("  action status: %v\n", step.Action.Status)
		writef("  compensation status: %v\n", step.Compensation.Status)
		writef("  compensation on action failure: %v\n", step.CompensationOnActionFailure)
		writef("  action calls: %d\n", step.Action.Calls)
		writef("  compensation calls: %d\n", step.Compensation.Calls)
		writef("-----\n")

		t.Logf("\n%s", sb.String())
	}
}
