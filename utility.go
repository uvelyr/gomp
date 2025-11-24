package main

import (
	"github.com/eiannone/keyboard"
)

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

type Command struct {
	Kind CommandKind
	Value any
}

type Song struct {
	Path string //path to a file, including the file itself
	Name string //name of a fle
}

type KeyPress struct {
	Rune rune
	Key keyboard.Key
}
