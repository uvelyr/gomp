package main

import (
	"slices"
	"fmt"
	"github.com/gdamore/tcell/v2"
)

type UIRequestType int

const (
	UIUpdatePlaybackBox UIRequestType = iota
	UIUpdatePlaylistBox
	UIDrawSongSelectionBox
	UIClearSongSelectionBox
	UISelectSong
)

type UIRequest struct {
	Type  UIRequestType
	Reply chan Song
}

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

func (ui *UI) clearBox(b Box) {
    for y := b.startY + 1; y < b.startY+b.height-1; y++ {
        for x := b.startX + 1; x < b.startX+b.width-1; x++ {
            ui.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
        }
    }
}

func (ui *UI) updateSongList(songs []Song, highlighted ...Song) {
	box := songListBox
	defaultStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	ui.drawBox(box)
	ui.clearBox(box)

	for i, song := range songs {
		y := box.startY + 1 + i
		if y >= box.startY+box.height-1 {
			break
		}

		style := defaultStyle.Reverse(slices.Contains(highlighted, song))

		line := song.Name

		for x, ch := range line {
			if x >= box.width-5 && len(line) > box.width-2{
				ui.screen.SetContent(box.startX+1+x, y, ch, nil, style)
				ui.screen.SetContent(box.startX+2+x, y, ch, nil, style)
				ui.screen.SetContent(box.startX+3+x, y, ch, nil, style)
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

func (ui *UI) drawPlaylistBox() {
	ui.drawBox(playlistBox)
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

//WIP
func (ui *UI) updatePlaylistBox() {
	ui.clearBox(playlistBox)
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	y := playlistBox.startY+1
	for _, song := range ui.playback.playlist {
		x := playlistBox.startX+1
		for i := 0; i < playlistBox.width-2 && i < len(song.Name); i++ {
			ui.screen.SetContent(x+i, y, rune(song.Name[i]), nil, style) 
		}
		y++
	}
	ui.screen.Show()
//	fmt.Println(ui.playback.playlist)
}

func (ui *UI) drawSongSearchBox() {
	box := songSearchBox
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)

	startX, startY, width, height := box.startX, box.startY, box.width, box.height
	h := '─'
	v := '│'
	bl := '└'
	br := '┘'

	title := " Search songs "
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

func (ui *UI) clearSongSearchBox() {
	box := songSearchBox
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

func (ui *UI) searchSongs(searchInput string) []Song{
	var matchingSongs []Song
	for _, song := range ui.allSongs {
		if len(searchInput) <= len(song.Name) && song.Name[:len(searchInput)] == searchInput {
			matchingSongs = append(matchingSongs, song)
		}
	}
	return matchingSongs
}

func (ui *UI) selectSong() Song {
	var searchInput []rune
	field := songSearchField

	updateSearchField := func() {
		// clear field
		for x := 0; x < field.width; x++ {
			ui.screen.SetContent(field.startX+x, field.startY, ' ', nil, tcell.StyleDefault)
		}
		// draw input
		for i, r := range searchInput {
			ui.screen.SetContent(field.startX+i, field.startY, r, nil, tcell.StyleDefault)
		}
		ui.screen.Show()
	}
	songs := ui.allSongs
	selectionID := -1
	if len(songs) != 0 {
		selectionID = 0
		ui.updateSongList(songs, songs[0])
	}
	for ev := range ui.songSelectionChan {
		switch ev.Key {
		case tcell.KeyDown:
			if selectionID+1 <= len(songs)-1 {
				selectionID++
			}
			ui.updateSongList(songs, songs[selectionID])
		case tcell.KeyUp:
			if selectionID-1 >= 0 {
				selectionID--
			}
			ui.updateSongList(songs, songs[selectionID])
		case tcell.KeyEnter:
			if selectionID >= 0 && selectionID <= len(songs)-1 {
				selectedSong := songs[selectionID]
				searchInput = nil
				songs = ui.allSongs
				ui.updateSongList(songs)
				updateSearchField()
				return selectedSong
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(searchInput) > 0 {
				searchInput = searchInput[:len(searchInput)-1]
				updateSearchField()
			}
			songs = ui.searchSongs(string(searchInput))
			if len(songs) == 0{
				selectionID = -1
			} else {
				ui.updateSongList(songs, songs[0])
			}
		case tcell.KeyEsc:
			//return empty to indicate that we're leaving song selection mode
			ui.updateSongList(songs)
			return Song{}
		case tcell.KeyRune:
			r := ev.Rune
			searchInput = append(searchInput, r)
			updateSearchField()
			songs = ui.searchSongs(string(searchInput))
			if len(songs) == 0{
				selectionID = -1
			}
			if selectionID >= 0 {
				ui.updateSongList(songs, songs[0])
			} else {
				ui.updateSongList(songs)
			}
		}
	}

	return Song{}
}

type UI struct {
	screen   tcell.Screen
	playback *PlaybackManager // Added: direct reference to player
	allSongs    []Song

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
func NewUI(screen tcell.Screen, playback *PlaybackManager, uiChan chan UIRequest, allSongs []Song) *UI {
	return &UI{
		screen:   screen,
		playback: playback, // Store playback reference
		allSongs:    allSongs,

		// UI creates and owns these channels
		songSelectionMode:   false,
		keyPressChan:        make(chan KeyPress, 32),
		songSelectionChan:   make(chan KeyPress, 32),
		updatePlaybackChan:  make(chan struct{}, 8),
		selectSongChan:      make(chan struct{}, 1),
		selectionResultChan: make(chan Song, 1),
		uiRequestChan:       uiChan,
		quitChan:            make(chan struct{}, 1),
	}
}

func (ui *UI) start() {
	ui.startRawInputLoop()
	ui.updateSongList(ui.allSongs)
	ui.drawPlaybackBox()
	ui.drawPlaylistBox()
	go ui.handleRequests()
}

func (ui *UI) handleRequests() {
	for r := range ui.uiRequestChan {
		switch r.Type {
		case UIUpdatePlaybackBox:
			ui.updatePlaybackBox()
		case UIUpdatePlaylistBox:
			ui.updatePlaylistBox()
		case UISelectSong:
			go func() {
				ui.drawSongSearchBox()
				selectedSong := ui.selectSong()
				ui.clearSongSearchBox()
				r.Reply <- selectedSong
				ui.updatePlaylistBox()
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
