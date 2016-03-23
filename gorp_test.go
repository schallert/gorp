package gorp

import (
	"math/rand"
	"testing"

	"github.com/schallert/gorp/rserve"
)

var vals [][2]float64

func init() {
	l := 48 * 24 * 60
	vals = make([][2]float64, l)
	for i := 0; i < l; i++ {
		vals[i] = [2]float64{float64(i), 100 + rand.Float64()*10}
	}
}

func BenchmarkMake(b *testing.B) {
	for j := 0; j < b.N; j++ {
		points := make([]rserve.Datapoint, len(vals))
		for i, v := range vals {
			points[i] = rserve.Datapoint{
				Timestamp: int64(v[0]),
				Value:     v[1],
			}
		}
	}
}

func BenchmarkAppend(b *testing.B) {
	for j := 0; j < b.N; j++ {
		var points []rserve.Datapoint
		for _, v := range vals {
			points = append(points, rserve.Datapoint{int64(v[0]), v[1]})
		}
	}
}
