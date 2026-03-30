package bench

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/kozmod/oniontx/saga"
)

func Benchmark_copy(b *testing.B) {
	const (
		letters   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		errorsLen = 100
	)
	var (
		generator = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
		genFn     = func() saga.ExecutionData {
			b.Helper()
			errors := make([]error, errorsLen)
			for i := 0; i < errorsLen; i++ {
				b := make([]byte, 10)
				for i := range b {
					b[i] = letters[generator.IntN(len(letters))]
				}
				errors[i] = fmt.Errorf("%s", string(b))
			}

			return saga.ExecutionData{
				Calls:  generator.Uint32(),
				Errors: errors,
				Status: saga.ExecutionStatusSuccess,
			}
		}

		slicesClone = func(in saga.ExecutionData) saga.ExecutionData {
			b.Helper()
			return saga.ExecutionData{
				Calls:  in.Calls,
				Errors: slices.Clone(in.Errors),
				Status: in.Status,
			}
		}

		oldCopy = func(in saga.ExecutionData) saga.ExecutionData {
			b.Helper()
			errors := make([]error, len(in.Errors))
			copy(errors, in.Errors)
			return saga.ExecutionData{
				Calls:  in.Calls,
				Errors: errors,
				Status: in.Status,
			}
		}
	)

	b.Run(fmt.Sprintf("size_%d", errorsLen), func(b *testing.B) {
		b.Run("copy", func(b *testing.B) {
			var (
				in  = genFn()
				res saga.ExecutionData
			)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				res = oldCopy(in)
			}

			_ = res
		})

		b.Run("slices.Clone", func(b *testing.B) {
			var (
				in  = genFn()
				res saga.ExecutionData
			)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				res = slicesClone(in)
			}

			_ = res
		})
	})

	b.Run("empty", func(b *testing.B) {
		b.Run("copy", func(b *testing.B) {
			var (
				in  = genFn()
				res saga.ExecutionData
			)

			b.ResetTimer()
			b.ReportAllocs()
			res.Errors = nil
			for i := 0; i < b.N; i++ {
				res = oldCopy(in)
			}

			_ = res
		})

		b.Run("slices.Clone", func(b *testing.B) {
			var (
				in  = genFn()
				res saga.ExecutionData
			)

			b.ResetTimer()
			b.ReportAllocs()
			res.Errors = nil
			for i := 0; i < b.N; i++ {
				res = slicesClone(in)
			}

			_ = res
		})
	})

	b.Run(fmt.Sprintf("empty_with_cap_%d", errorsLen), func(b *testing.B) {
		b.Run("copy", func(b *testing.B) {

			var (
				in  = genFn()
				res saga.ExecutionData
			)
			res.Errors = nil

			b.ResetTimer()
			b.ReportAllocs()
			res.Errors = make([]error, cap(in.Errors))
			for i := 0; i < b.N; i++ {
				res = oldCopy(in)
			}

			_ = res
		})

		b.Run("slices.Clone", func(b *testing.B) {
			var (
				in  = genFn()
				res saga.ExecutionData
			)
			b.ResetTimer()
			b.ReportAllocs()
			res.Errors = make([]error, cap(in.Errors))
			for i := 0; i < b.N; i++ {
				res = slicesClone(in)
			}

			_ = res
		})
	})

}
