package examples

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kozmod/oniontx/saga"
)

// Test_Saga_example demonstrates two approaches to creating sagas:
//
// 1. Simple builder style — for straightforward, linear processes
//
// 2. Advanced builder style — for complex scenarios with retry and recovery
func Test_Saga_example(t *testing.T) {
	t.Skip()

	var (
		// ErrPaymentFailed is an example error type for demonstration.
		ErrPaymentFailed = fmt.Errorf("payment failed")

		// refundPayment is an example compensation function.
		refundPayment = func(ctx context.Context) error {
			// Implementation would refund a payment.
			return nil
		}
	)

	t.Run("first_example: simple builder approach", func(t *testing.T) {
		// Create saga steps with the public builder API.
		// This approach is ideal when:
		// - You have simple actions and compensation logic
		// - You want maximum readability
		// - You don't need additional decorators (retry, panic recovery)
		steps := []saga.Step{
			saga.NewStep("first_step").
				WithAction(saga.NewOperation(
					// Action — the main function of the step
					// Executes business logic and returns an error on failure
					// The track parameter provides access to execution context:
					//   - track.GetStepData() — retrieve step execution data
					//   - track.SetStatus(status) — update execution status
					//   - track.AddError(err) — append errors without changing status
					func(ctx context.Context, track saga.Track) error {
						// This could be:
						// - Database query via mtx.Transactor
						// - External API call
						// - Any other operation
						//
						// Use track to record intermediate errors:
						// if err := someOperation(ctx); err != nil {
						//     track.SetStatus(saga.ExecutionStatusFail)
						//     track.AddError(err)
						//     return err
						// }
						return nil
					})).
				// Compensation — rollback function
				// Called if subsequent steps fail
				// Important: compensation must be idempotent!
				// The track contains information about the failed action:
				//   - track.GetStepData().Action.Errors — errors from the action
				//   - track.GetStepData().Action.Status — status of the action
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					// Get execution data to make decisions based on error type
					data := track.GetStepData()

					// Example: conditional compensation based on error
					if len(data.Action.Errors) > 0 {
						if errors.Is(data.Action.Errors[0], ErrPaymentFailed) {
							// Handle specific error type
							return refundPayment(ctx)
						}
					}

					// Default compensation logic
					return nil
				})).
				// WithCompensationRequired determines whether this step can compensate
				// its own action failure. Without it, only successfully completed
				// steps are compensated after a later step fails.
				WithCompensationRequired(),
		}

		// Create and execute the Saga.
		// Saga automatically manages the order: actions execute sequentially,
		// on error, compensations run in reverse order
		result, err := saga.NewSaga(steps).Execute(context.Background())

		// Handle the result
		if err != nil {
			// err contains detailed information about failures
			// Use result to get detailed step-by-step execution data
			t.Logf("Saga failed: %v\n", err)
			fmt.Printf("Result status: %s\n", result.Status)
		}
	})

	t.Run("second_example: advanced approach with builder", func(t *testing.T) {
		// Use the builder API for more complex configuration.
		// This approach provides access to all library features:
		// - Panic recovery
		// - Retry policies
		// - Custom backoff strategies
		// - Jitter for load distribution
		steps := []saga.Step{
			saga.NewStep("first_step").
				WithAction(
					// Add action with decorators
					saga.NewOperation(func(ctx context.Context, track saga.Track) error {
						// Simulate error to demonstrate retry
						// Record the error in track
						err := fmt.Errorf("first_step_Error")
						track.SetStatus(saga.ExecutionStatusFail)
						// Add the error to the errors list.
						track.AddError(err)
						return err
					}).
						// Protection against panics — important for production!
						// If the action panics, the panic will be caught
						// and returned as an error with ErrPanicRecovered
						WithPanicRecovery().
						// Add retry for action
						WithRetry(
							// 2 attempts, 1s between attempts
							saga.NewBaseRetryOpt(2, 1*time.Second),
						),
				).
				// Add compensation
				WithCompensation(
					saga.NewOperation(func(ctx context.Context, track saga.Track) error {
						// Compensation logic.
						// Get data to understand what failed
						data := track.GetStepData()

						// Log the error that triggered compensation
						if len(data.Action.Errors) > 0 {
							fmt.Printf("Compensating for error: %v\n", data.Action.Errors[0])
						}

						// Perform compensation
						return nil
					}).
						// Compensation can also have retry logic
						WithRetry(
							saga.NewAdvanceRetryPolicy(
								2,                            // max attempts
								1*time.Second,                // initial delay
								saga.NewExponentialBackoff(), // exponential backoff
							).
								// Jitter prevents "thundering herd"
								WithJitter(
									// random delay
									saga.NewFullJitter(),
								).
								// maximum delay
								WithMaxDelay(10 * time.Second),
						),
				).
				// Mark that this step requires compensation
				WithCompensationRequired(),
		}

		// Execute the saga
		//
		// With this approach:
		// 1. If action fails, there will be 2 attempts with fixed delay
		// 2. If all attempts fail, compensations will run
		// 3. Compensations will also retry on failure with exponential backoff
		// 4. Jitter distributes load during mass failures
		result, err := saga.NewSaga(steps).Execute(context.Background())

		if err != nil {
			// Handle error with full context.
			fmt.Printf("Saga execution failed: %v\n", err)
			fmt.Printf("Result status: %s\n", result.Status)
		}
	})
}
