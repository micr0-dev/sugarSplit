package main

import "time"

var bigNumbers = [][]string{
	{ // 0
		"█▀█",
		"█ █",
		"▀▀▀",
	},
	{ // 1
		"▄█ ",
		" █ ",
		"▀▀▀",
	},
	{ // 2
		"▀▀█",
		"█▀▀",
		"▀▀▀",
	},
	{ // 3
		"▀▀█",
		"▀▀█",
		"▀▀▀",
	},
	{ // 4
		"█ █",
		"▀▀█",
		"  ▀",
	},
	{ // 5
		"█▀▀",
		"▀▀█",
		"▀▀▀",
	},
	{ // 6
		"█▀▀",
		"█▀█",
		"▀▀▀",
	},
	{ // 7
		"█▀█",
		"  █",
		"  ▀",
	},
	{ // 8
		"█▀█",
		"█▀█",
		"▀▀▀",
	},
	{ // 9
		"█▀█",
		"▀▀█",
		"  ▀",
	},
}

func getBigNumber(n int) []string {
	if n < 0 || n > 9 {
		return []string{"   ", "   ", "   "}
	}
	return bigNumbers[n]
}

func getBigColon() []string {
	return []string{
		" ▀ ",
		"   ",
		" ▀ ",
	}
}

func getBigDot() []string {
	return []string{
		"   ",
		"   ",
		" ▀ ",
	}
}

func getBigTimer(d time.Duration) []string {
	d = d.Round(time.Millisecond)
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000 / 10 // Get centiseconds (2 digits)

	min1 := minutes / 10
	min2 := minutes % 10
	sec1 := seconds / 10
	sec2 := seconds % 10
	ms1 := milliseconds / 10
	ms2 := milliseconds % 10

	result := make([]string, 3)

	// Combine each row
	for i := 0; i < 3; i++ {
		result[i] = getBigNumber(min1)[i] + "  " + getBigNumber(min2)[i] +
			getBigColon()[i] +
			getBigNumber(sec1)[i] + "  " + getBigNumber(sec2)[i] +
			getBigDot()[i] +
			getBigNumber(ms1)[i] + "  " + getBigNumber(ms2)[i]
	}

	return result
}
