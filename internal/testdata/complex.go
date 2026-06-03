package testdata

func veryComplex(x int) int {
	if x > 0 {
		if x > 10 {
			if x > 20 {
				if x > 30 {
					if x > 40 {
						if x > 50 {
							if x > 60 {
								if x > 70 {
									return 8
								}
								return 7
							}
							return 6
						}
						return 5
					}
					return 4
				}
				return 3
			}
			return 2
		}
		return 1
	}
	return 0
}

func withSwitch(x int) int {
	switch x {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	default:
		return 0
	}
}
