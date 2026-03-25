package saga

import (
	"fmt"
	"testing"

	"github.com/kozmod/oniontx/saga"
)

func printResult(t *testing.T, res saga.Result, err error) {
	t.Helper()
	t.Logf("\nresult:\n%v", res)
	fmt.Printf("\nexecution error: %v\n", err)

	for _, step := range res.Steps {
		fmt.Printf("-----")
		fmt.Printf("\nstep [%d#%s]:\n", step.StepPosition, step.StepName)

		switch {
		case len(step.Action.Errors) > 0:
			fmt.Printf("  action errors (%d):\n", len(step.Action.Errors))
			for i, e := range step.Action.Errors {
				fmt.Printf("    %d: %v\n", i, e)
			}
		default:
			fmt.Printf("  action errors: none\n")
		}

		switch {
		case len(step.Compensation.Errors) > 0:
			fmt.Printf("  compensation errors (%d):\n", len(step.Compensation.Errors))
			for i, e := range step.Compensation.Errors {
				fmt.Printf("    %d: %v\n", i, e)
			}
		default:
			fmt.Printf("  compensation errors: none\n")
		}

		fmt.Printf("  -----\n")
		fmt.Printf("  action status: %v\n", step.Action.Status)
		fmt.Printf("  compensation status: %v\n", step.Compensation.Status)
		fmt.Printf("  compensation required: %v\n", step.CompensationRequired)
		fmt.Printf("  action calls: %d\n", step.Action.Calls)
		fmt.Printf("  compensation calls: %d\n", step.Compensation.Calls)
		fmt.Printf("-----\n")
	}
}
