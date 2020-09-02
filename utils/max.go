package utils

import "github.com/256dpi/max-go"

// Int will return any number as an int.
func Int(atom max.Atom) int64 {
	switch v := atom.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// Float will return any number as a float.
func Float(atom max.Atom) float64 {
	switch v := atom.(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}
