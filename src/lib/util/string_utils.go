package util

// ConditionalString returns valueIfTrue if condition is true, otherwise valueIfFalse
func ConditionalString(condition bool, valueIfTrue, valueIfFalse string) string {
	if condition {
		return valueIfTrue
	}
	return valueIfFalse
}