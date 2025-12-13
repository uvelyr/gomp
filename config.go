package main

import (
	"time"
)

const (
	bufferSize = time.Second / 10
	speedStep  = 0.05
	volumeStep = 0.35
	seekStep   = time.Second * 5
	minVolume  = -5.0
	maxVolume  = 2.0
	minSpeed   = 0.6
	maxSpeed   = 2.0
)

var (
	songListBox        = Box{0, 0, 25, 20}
	songSelectionBox   = Box{songListBox.startX, songListBox.startY + songListBox.height, songListBox.width, 2}
	songSelectionField = Field{songSelectionBox.startX + 1, songSelectionBox.startY, songSelectionBox.width - 2, 2}
	playbackBox        = Box{songListBox.startX + songListBox.width + 1, 0, 25, 7}
)
