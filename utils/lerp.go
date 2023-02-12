package utils

import "github.com/go-gl/mathgl/mgl64"

// Lerp will perform a linear interpolation on the provided values.
func Lerp(v1, v2, t float64) float64 {
	return v1*(1-t) + v2*t
}

// LerpVec3 will perform a linear interpolation on the provided vectors.
func LerpVec3(v1, v2 mgl64.Vec3, t float64) mgl64.Vec3 {
	return mgl64.Vec3{
		Lerp(v1.X(), v2.X(), t),
		Lerp(v1.Y(), v2.Y(), t),
		Lerp(v1.Z(), v2.Z(), t),
	}
}
