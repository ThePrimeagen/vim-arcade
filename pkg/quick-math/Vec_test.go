package quickmath_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"vim-arcade.theprimeagen.com/pkg/quick-math"
)

var Vec = quickmath.NewVec2
func TestVec2Init(t *testing.T) {
    vec := Vec(1.0, 2.0)
    require.Equal(t, vec, Vec(1.0, 2.0))
}

func TestVec2Operations(t *testing.T) {
    vec := Vec(1.0, 2.0)
    vecLen := math.Sqrt(1 + 4)
    require.Equal(t, vec.Add(Vec(68.0, 67.0)), Vec(69, 69))
    require.Equal(t, vec.Mul(Vec(3.5, 4.345)), Vec(3.5, 8.69))
    require.Equal(t, vec.Scale(4), Vec(4, 8))
    require.Equal(t, vec.Sub(Vec(4, 3.5)), Vec(-3.0, -1.5))
    require.Equal(t, vec.Len(), vecLen)
    require.Equal(t, Vec(0, 0).Len(), 0.0)
    require.Equal(t, Vec(0, 0).Norm(), Vec(0, 0))
    require.Equal(t, vec.LenSq(), 5.0)

    require.Equal(t, vec.Norm(), Vec(1.0 / vecLen, 2.0 / vecLen))
}


