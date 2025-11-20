package main

func clampFloat(v, lo, hi float64) float64 {
	if v < lo { return lo }
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo { return lo }
	if v > hi { return hi }
	return v
}
