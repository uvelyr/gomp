package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/gopxl/beep"
)

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// converts sample count to mm:ss string
func formatDuration(pos int, sr beep.SampleRate) string {
	secs := int(float64(pos) / float64(sr))
	min := secs / 60
	sec := secs % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

type Command struct {
	Kind  CommandKind
	Value any
}

type Song struct {
	Path string //path to a file, including the file itself
	Name string //name of a fle
}

type KeyPress struct {
	Rune rune
	Key  tcell.Key
}
