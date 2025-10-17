package sage

import (
	"context"
	"log"
	"testing"
)

func Test(t *testing.T) {
	s := Stage{}
	var i int32
	s.AddStage(func(ctx context.Context) error {
		i = 99
		return nil
	})
	s.AddStage(func(ctx context.Context) error {
		log.Print(i)
		return nil
	})
	s.Exec(context.Background())
}
