package quickmath

type AABB struct {
	Min, Max Vec2
}

func (a AABB) Intersect(b AABB) bool {
	return a.Min.X < b.Max.X && a.Max.X > b.Min.X &&
		a.Min.Y < b.Max.Y && a.Max.Y > b.Min.Y
}

