package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"strconv"
)

type UIRequest int

const (
	UIUpdatePlaybackBox UIRequest = iota
	UIDrawSongSelectionBox
	UIClearSongSelectionBox
	UISelectSong
)

func (ui *UI) drawBox(box Box) {
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

	startX, startY, width, height := box.startX, box.startY, box.width, box.height
	h := '─'
	v := '│'
	tl := '┌'
	tr := '┐'
	bl := '└'
	br := '┘'

	ui.screen.SetContent(startX, startY, tl, nil, style)
	ui.screen.SetContent(startX+width-1, startY, tr, nil, style)
	ui.screen.SetContent(startX, startY+height-1, bl, nil, style)
	ui.screen.SetContent(startX+width-1, startY+height-1, br, nil, style)

	for x := startX + 1; x < startX+width-1; x++ {
		ui.screen.SetContent(x, startY, h, nil, style)
		ui.screen.SetContent(x, startY+height-1, h, nil, style)
	}

	for y := startY + 1; y < startY+height-1; y++ {
		ui.screen.SetContent(startX, y, v, nil, style)
		ui.screen.SetContent(startX+width-1, y, v, nil, style)
	}
}

func (ui *UI) displaySongList() {
	box := songListBox
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

	ui.drawBox(songListBox)

	for i, song := range ui.songs {
		y := box.startY + 1 + i
		if y >= box.startY+box.height-1 {
			break
		}
		line := strconv.Itoa(i+1) + ": " + song.Name
		for x, ch := range line {
			if x >= box.width-2 {
				break
			}
			ui.screen.SetContent(box.startX+1+x, y, ch, nil, style)
		}
	}

	ui.screen.Show()
}

func (ui *UI) drawPlaybackBox() {
	ui.drawBox(playbackBox)
	ui.screen.Show()
}

func (ui *UI) updatePlaybackBox() {
	pm, box, p := ui.playback, playbackBox, ui.playback.player

	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

	// Coordinates inside the box
	x := box.startX + 1
	y := box.startY + 1
	width := box.width - 2

	// Current song
	var songName string
	if pm.currentSong != nil {
		songName = pm.currentSong.Name
	} else {
		songName = "No song"
	}
	if len(songName) > width {
		songName = songName[:width-1] + "…" // truncate with ellipsis
	}

	// Clear song name field first
	for i := 0; i < width; i++ {
		ui.screen.SetContent(x+i, y, ' ', nil, style)
	}
	// Draw song name centered
	songX := x + (width-len(songName))/2
	for i, r := range songName {
		ui.screen.SetContent(songX+i, y, r, nil, style)
	}

	// Status
	status := "⏹" // stopped
	if p.Control != nil {
		if p.Control.Paused {
			status = "⏸"
		} else {
			status = "▶"
		}
	}
	statusStr := "Status: " + status
	for i, r := range statusStr {
		ui.screen.SetContent(x+i, y+1, r, nil, style)
	}

	// Speed
	speedVal := 1.0
	if p.Speed != nil {
		speedVal = p.Speed.Ratio()
	}
	speedStr := fmt.Sprintf("Speed: %.2fx", speedVal)
	for i, r := range speedStr {
		ui.screen.SetContent(x+i, y+2, r, nil, style)
	}

	// Volume
	vol := 0.0
	if p.Volume != nil {
		vol = p.Volume.Volume
	}
	volStr := fmt.Sprintf("Vol: %.1f dB", vol)
	for i, r := range volStr {
		ui.screen.SetContent(x+i, y+3, r, nil, style)
	}

	// Elapsed / total length
	var elapsed, totalSamples int
	if p.Streamer != nil {
		elapsed = p.Streamer.Position()
		totalSamples = p.Streamer.Len()
	}
	elapsedTime := formatDuration(elapsed, p.Format.SampleRate)
	totalTime := formatDuration(totalSamples, p.Format.SampleRate)
	timeStr := fmt.Sprintf("Time: %s / %s", elapsedTime, totalTime)
	for i, r := range timeStr {
		ui.screen.SetContent(x+i, y+4, r, nil, style)
	}

	ui.screen.Show()
}

func (ui *UI) drawSongSelectionBox() {
	box := songSelectionBox
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

	startX, startY, width, height := box.startX, box.startY, box.width, box.height
	h := '─'
	v := '│'
	bl := '└'
	br := '┘'

	title := "Enter song id"
	titleRunes := []rune(title)
	titleLen := len(titleRunes)

	// left and right junctions
	ui.screen.SetContent(startX, startY-1, '├', nil, style)
	ui.screen.SetContent(startX+width-1, startY-1, '┤', nil, style)

	// horizontal line with text in the middle
	textStart := startX + (width-titleLen-2)/2 // -2 for spacing before and after
	for x := startX + 1; x < startX+width-1; x++ {
		switch {
		case x == textStart-1 || x == textStart+titleLen:
			// optional spacing around text
			ui.screen.SetContent(x, startY-1, h, nil, style)
		case x >= textStart && x < textStart+titleLen:
			ui.screen.SetContent(x, startY-1, titleRunes[x-textStart], nil, style)
		default:
			ui.screen.SetContent(x, startY-1, h, nil, style)
		}
	}

	// bottom corners
	ui.screen.SetContent(startX, startY+height-1, bl, nil, style)
	ui.screen.SetContent(startX+width-1, startY+height-1, br, nil, style)

	// bottom horizontal line
	for x := startX + 1; x < startX+width-1; x++ {
		ui.screen.SetContent(x, startY+height-1, h, nil, style)
	}

	// vertical sides
	for y := startY; y < startY+height-1; y++ {
		ui.screen.SetContent(startX, y, v, nil, style)
		ui.screen.SetContent(startX+width-1, y, v, nil, style)
	}

	ui.screen.Show()
}

func (ui *UI) clearSongSelectionBox() {
	box := songSelectionBox
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

	startX, startY, width, height := box.startX, box.startY, box.width, box.height

	for y := startY; y < startY+height; y++ {
		for x := startX; x < startX+width; x++ {
			ui.screen.SetContent(x, y, ' ', nil, style)
		}
	}

	//return last box line of songList to normal condition
	for x := startX + 1; x < startX+width-1; x++ {
		ui.screen.SetContent(x, startY-1, '─', nil, style)
	}
	ui.screen.SetContent(startX, startY-1, '└', nil, style)
	ui.screen.SetContent(startX+width-1, startY-1, '┘', nil, style)
}

func (ui *UI) selectSong() Song {
	var input []rune
	field := songSelectionField

	updateInputField := func() {
		// clear field
		for x := 0; x < field.width; x++ {
			ui.screen.SetContent(field.startX+x, field.startY, ' ', nil, tcell.StyleDefault)
		}

		// draw digits
		for i, r := range input {
			ui.screen.SetContent(field.startX+i, field.startY, r, nil, tcell.StyleDefault)
		}

		ui.screen.Show()
	}

	for ev := range ui.songSelectionChan {
		switch ev.Key {

		case tcell.KeyEnter:
			if len(input) == 0 {
				input = nil
				updateInputField()
				return Song{}
			}
			n, err := strconv.Atoi(string(input))
			if err != nil || n < 1 || n > len(ui.songs) {
				msg := "Invalid id"
				style := tcell.StyleDefault.Foreground(tcell.ColorGray)
				for i, r := range msg {
					ui.screen.SetContent(field.startX+i, field.startY, r, nil, style)
				}
				ui.screen.Show()

				input = nil
				updateInputField()
				continue
			}
			return ui.songs[n-1]
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(input) > 0 {
				input = input[:len(input)-1]
				updateInputField()
			}
		case tcell.KeyEsc:
			//return empty to indicate that we're leaving song selection mode
			return Song{}

		case tcell.KeyRune:
			r := ev.Rune
			if r >= '0' && r <= '9' {
				input = append(input, r)
				updateInputField()
			}
		}
	}

	return Song{}
}

type UI struct {
	screen   tcell.Screen
	playback *PlaybackManager // Added: direct reference to player
	songs    []Song

	//UI OWNS these channels (not read-only in struct)
	songSelectionMode   bool
	keyPressChan        chan KeyPress // Produced by UI, consumed by CommandHandler
	songSelectionChan   chan KeyPress
	selectionResultChan chan Song // Result of song selection
	uiRequestChan       chan UIRequest
	updatePlaybackChan  chan struct{} // Requests to update playback display
	selectSongChan      chan struct{} // Requests to choose song
	quitChan            chan struct{} // Signal to stop UI
}

// Constructor returns UI with all channels it owns
func NewUI(screen tcell.Screen, playback *PlaybackManager, songs []Song) *UI {
	return &UI{
		screen:   screen,
		playback: playback, // Store playback reference
		songs:    songs,

		// UI creates and owns these channels
		songSelectionMode:   false,
		keyPressChan:        make(chan KeyPress, 32),
		songSelectionChan:   make(chan KeyPress, 32),
		updatePlaybackChan:  make(chan struct{}, 8),
		selectSongChan:      make(chan struct{}, 1),
		selectionResultChan: make(chan Song, 1),
		uiRequestChan:       make(chan UIRequest),
		quitChan:            make(chan struct{}, 1),
	}
}

func (ui *UI) start() {
	ui.startRawInputLoop()
	ui.displaySongList()
	ui.drawPlaybackBox()
	go ui.handleRequests()
}

func (ui *UI) handleRequests() {
	for r := range ui.uiRequestChan {
		switch r {
		case UIUpdatePlaybackBox:
			ui.updatePlaybackBox()
		case UISelectSong:
			go func() {
				ui.drawSongSelectionBox()
				selectedSong := ui.selectSong()
				ui.clearSongSelectionBox()
				ui.selectionResultChan <- selectedSong
				ui.screen.Show()
			}()
		}
	}
}

func (ui *UI) startRawInputLoop() {
	go func() {
		for {
			ev := ui.screen.PollEvent()

			switch e := ev.(type) {
			case *tcell.EventKey:
				ui.keyPressChan <- KeyPress{Rune: e.Rune(), Key: e.Key()}
			case *tcell.EventResize:
			}
		}
	}()
}

//must handle
