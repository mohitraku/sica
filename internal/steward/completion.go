package steward

func IsDone(value, target int) bool {
	return value >= target
}

func IncrementValue(current, target int) int {
	if current >= target {
		return current
	}
	return current + 1
}

func DecrementValue(current int) int {
	if current <= 0 {
		return 0
	}
	return current - 1
}
