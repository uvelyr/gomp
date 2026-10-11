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

//box constructor arg order is startx starty width height
var (
	songListBox        = Box{0, 0, 25, 20}
	songSearchBox   = Box{songListBox.startX, songListBox.startY + songListBox.height, songListBox.width, 2}
	songSearchField = Field{songSearchBox.startX + 1, songSearchBox.startY, songSearchBox.width - 2, 2}
	playbackBox        = Box{songListBox.startX + songListBox.width + 1, 0, 25, 7}
	playlistBox = Box{songListBox.startX + songListBox.width + 1, playbackBox.startY + playbackBox.height, playbackBox.width, songListBox.height - playbackBox.height} 
)
