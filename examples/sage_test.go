package examples

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kozmod/oniontx/saga"
)

// Test_Saga_example demonstrates two approaches to creating sagas:
//
// 1. Simple, declarative — for straightforward, linear processes
//
// 2. Advanced, using builder — for complex scenarios with retry and recovery
func Test_Saga_example(t *testing.T) {
	t.Skip()

	t.Run("first_example: simple declarative approach", func(t *testing.T) {
		// Create saga steps as simple structs
		// This approach is ideal when:
		// - You have simple actions, compensation and retries logic
		// - You want maximum readability
		// - You don't need additional decorators
		steps := []saga.Step{
			{
				// Name is used for logging and debugging
				// Best practice: give meaningful names
				Name: "first_step",

				// Action — the main function of the step
				// Executes business logic and returns an error on failure
				Action: func(ctx context.Context) error {
					// This could be:
					// - Database query via mtx.Transactor
					// - External API call
					// - Any other operation
					return nil
				},

				// Compensation — rollback function
				// Called if subsequent steps fail
				// Important: compensation must be idempotent!
				Compensation: func(ctx context.Context, aroseErr error) error {
					// aroseErr — the error that triggered compensation
					// This allows making different decisions based on error type

					// Example: if errors.Is(aroseErr, ErrPaymentFailed) {
					//     return refundPayment(ctx)
					// }
					return nil
				},

				// CompensationOnFail determines whether this step needs compensation
				// true: if step changes state and requires rollback
				// false: for read-only operations or non-compensatable actions (email, notifications)
				CompensationOnFail: true,
			},
		}

		// Create and execute the saga
		// Saga automatically manages the order: actions execute sequentially,
		// on error, compensations run in reverse order
		err := saga.NewSaga(steps).Execute(context.Background())

		// Handle the error
		if err != nil {
			// Important: err may contain both execution errors and compensation errors
		}
	})

	t.Run("second_example: advanced approach with builder", func(t *testing.T) {
		// Use StepBuilder for more complex configuration
		// This approach provides access to all library features:
		// - Panic recovery
		// - Retry policies
		// - Custom backoff strategies
		// - Jitter for load distribution
		steps := []saga.Step{
			saga.NewStep("first_step").
				WithAction(
					// Add action with decorators
					saga.NewAction(func(ctx context.Context) error {
						// Simulate error to demonstrate retry
						return fmt.Errorf("first_step_Error")
					}).
						// Protection against panics — important for production!
						// If the action panics, the panic will be caught
						// and returned as an error with ErrPanicRecovered
						WithPanicRecovery().
						// Add retry for action
						WithRetry(
							// 2 attempts, 1s between attempts
							saga.NewBaseRetryOpt(2, 1*time.Second).
								// Return all errors  which arise during retries
								WithReturnAllAroseErr(),
						),
				).
				// Add compensation
				WithCompensation(
					saga.NewCompensation(func(ctx context.Context, aroseErr error) error {
						// Compensation logic.
						// aroseErr — error from action that triggered compensation
						// This can be useful for logging or strategy selection
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
				),
		}

		// Execute the saga
		//
		// With this approach:
		// 1. If action fails, there will be 2 attempts with exponential backoff
		// 2. If all attempts fail, compensations will run
		// 3. Compensations will also retry on failure
		// 4. Jitter distributes load during mass failures
		err := saga.NewSaga(steps).Execute(context.Background())

		if err != nil {
			// Handle error
		}
	})
}
