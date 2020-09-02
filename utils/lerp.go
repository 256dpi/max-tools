package utils

// Lerp will perform a linear interpolation on the provided values.
func Lerp(v1, v2, t float64) float64 {
	return v1*(1-t) + v2*t
}
