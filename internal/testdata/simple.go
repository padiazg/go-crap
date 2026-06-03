package testdata

func simple() int {
	return 42
}

func withIf(x int) int {
	if x > 0 {
		return 1
	}
	return 0
}

func complex(x int) int {
	if x > 0 {
		if x > 10 {
			if x > 100 {
				return 3
			}
			return 2
		}
		return 1
	}
	return 0
}
