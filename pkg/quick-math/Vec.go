package quickmath

import "math"

type Vec2 struct {
	X, Y float64
}

func NewVec2(x, y float64) Vec2 {
	return Vec2{X: x, Y: y}
}

func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vec2) Mul(other Vec2) Vec2 {
	return Vec2{X: v.X * other.X, Y: v.Y * other.Y}
}

func (v Vec2) Scale(scalar float64) Vec2 {
	return Vec2{X: v.X * scalar, Y: v.Y * scalar}
}

func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vec2) Len() float64 {
	return math.Sqrt(v.LenSq())
}

func (v Vec2) LenSq() float64 {
	return v.X*v.X + v.Y*v.Y
}

func (v Vec2) Norm() Vec2 {
	length := v.Len()
	if length == 0 {
		return Vec2{X: 0, Y: 0}
	}
	return Vec2{X: v.X / length, Y: v.Y / length}
}

