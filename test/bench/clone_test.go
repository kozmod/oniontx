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
		ErrorsLen = 100
	)
	var (
		generator = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
		genFn     = func() saga.TrackData {
			b.Helper()
			Errors := make([]error, ErrorsLen)
			for i := 0; i < ErrorsLen; i++ {
				b := make([]byte, 10)
				for i := range b {
					b[i] = letters[generator.IntN(len(letters))]
				}
				Errors[i] = fmt.Errorf("%s", string(b))
			}

			return saga.TrackData{
				Calls:  generator.Uint32(),
				Errors: Errors,
				Status: saga.ExecutionStatusSuccess,
			}
		}

		slicesClone = func(in saga.TrackData) saga.TrackData {
			b.Helper()
			return saga.TrackData{
				Calls:  in.Calls,
				Errors: slices.Clone(in.Errors),
				Status: in.Status,
			}
		}

		oldCopy = func(in saga.TrackData) saga.TrackData {
			b.Helper()
			Errors := make([]error, len(in.Errors))
			copy(Errors, in.Errors)
			return saga.TrackData{
				Calls:  in.Calls,
				Errors: Errors,
				Status: in.Status,
			}
		}
	)

	b.Run(fmt.Sprintf("size_%d", ErrorsLen), func(b *testing.B) {
		b.Run("copy", func(b *testing.B) {
			var (
				in  = genFn()
				res saga.TrackData
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
				res saga.TrackData
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
				res saga.TrackData
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
				res saga.TrackData
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

	b.Run(fmt.Sprintf("empty_with_cap_%d", ErrorsLen), func(b *testing.B) {
		b.Run("copy", func(b *testing.B) {

			var (
				in  = genFn()
				res saga.TrackData
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
				res saga.TrackData
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
