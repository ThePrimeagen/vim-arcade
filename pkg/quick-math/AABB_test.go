package quickmath_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	quickmath "vim-arcade.theprimeagen.com/pkg/quick-math"
)

type Vec2 = quickmath.Vec2
type AABB = quickmath.AABB

func TestAABBIntersect(t *testing.T) {
	// Positive Cases (Intersections)
	t.Run("Basic intersection", func(t *testing.T) {
		a := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 5, Y: 5}}
		b := AABB{Min: Vec2{X: 3, Y: 3}, Max: Vec2{X: 7, Y: 7}}
		require.True(t, a.Intersect(b), "Expected AABBs to intersect")
		require.True(t, b.Intersect(a), "Expected AABBs to intersect")
	})

	t.Run("Exact overlap", func(t *testing.T) {
		a := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 5, Y: 5}}
		b := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 5, Y: 5}}
		require.True(t, a.Intersect(b), "Expected AABBs to intersect")
	})

	t.Run("One inside the other", func(t *testing.T) {
		a := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 10, Y: 10}}
		b := AABB{Min: Vec2{X: 3, Y: 3}, Max: Vec2{X: 7, Y: 7}}
		require.True(t, a.Intersect(b), "Expected AABBs to intersect")
		require.True(t, b.Intersect(a), "Expected AABBs to intersect")
	})

	// Negative Cases (No Intersection)
	t.Run("No intersection - completely separate UD", func(t *testing.T) {
		a := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 5, Y: 5}}
		b := AABB{Min: Vec2{X: 0, Y: 6}, Max: Vec2{X: 5, Y: 10}}
		require.False(t, a.Intersect(b), "Expected AABBs to not intersect")
		require.False(t, b.Intersect(a), "Expected AABBs to not intersect")
	})

	t.Run("No intersection - completely separate LR", func(t *testing.T) {
		a := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 5, Y: 5}}
		b := AABB{Min: Vec2{X: 5, Y: 0}, Max: Vec2{X: 10, Y: 5}}
		require.False(t, a.Intersect(b), "Expected AABBs to not intersect")
		require.False(t, b.Intersect(a), "Expected AABBs to not intersect")
	})

	t.Run("Touching edges - no intersection LR", func(t *testing.T) {
		a := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 5, Y: 5}}
		b := AABB{Min: Vec2{X: 5, Y: 0}, Max: Vec2{X: 10, Y: 5}}
		require.False(t, a.Intersect(b), "Expected AABBs to not intersect")
		require.False(t, b.Intersect(a), "Expected AABBs to not intersect")
	})

	t.Run("Touching edges - no intersection UD", func(t *testing.T) {
		a := AABB{Min: Vec2{X: 0, Y: 5}, Max: Vec2{X: 5, Y: 10}}
		b := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 5, Y: 5}}
		require.False(t, a.Intersect(b), "Expected AABBs to not intersect")
		require.False(t, b.Intersect(a), "Expected AABBs to not intersect")
	})

	t.Run("Touching corners - no intersection", func(t *testing.T) {
		a := AABB{Min: Vec2{X: 0, Y: 0}, Max: Vec2{X: 5, Y: 5}}
		b := AABB{Min: Vec2{X: 5, Y: 5}, Max: Vec2{X: 10, Y: 10}}
		require.False(t, a.Intersect(b), "Expected AABBs to not intersect")
		require.False(t, b.Intersect(a), "Expected AABBs to not intersect")
	})
}


