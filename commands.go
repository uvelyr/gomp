package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"sync"
	"time"
)

type CommandKind int

const (
	CmdAddToQueue CommandKind = iota
	CmdSkip
	CmdShowQueue
	CmdSeek
	CmdTogglePause
	CmdChangeSpeed
	CmdChangeVolume
	CmdQuit
	CmdNone
)

func translateRune(r rune) Command {
	switch r {
	case 'q', 'Q':
		return Command{Kind: CmdAddToQueue}
	case 's', 'S':
		return Command{Kind: CmdSkip}
	case 'd', 'D':
		return Command{Kind: CmdShowQueue}
	case 'n', 'N':
		return Command{Kind: CmdChangeSpeed, Value: -speedStep}
	case 'm', 'M':
		return Command{Kind: CmdChangeSpeed, Value: speedStep}
	case ' ':
		return Command{Kind: CmdTogglePause}
	default:
		return Command{Kind: CmdNone}
	}
}

func translateKey(k tcell.Key) Command {
	switch k {
	case tcell.KeyLeft:
		return Command{Kind: CmdSeek, Value: -seekStep}
	case tcell.KeyRight:
		return Command{Kind: CmdSeek, Value: seekStep}
	case tcell.KeyDown:
		return Command{Kind: CmdChangeVolume, Value: -volumeStep}
	case tcell.KeyUp:
		return Command{Kind: CmdChangeVolume, Value: volumeStep}
	case tcell.KeyEsc:
		return Command{Kind: CmdQuit}
	default:
		return Command{Kind: CmdNone}
	}
}

func translateKeyPress(kp KeyPress) Command {
	if kp.Rune != 0 {
		return translateRune(kp.Rune)
	} else {
		return translateKey(kp.Key)
	}
}

type CommandHandler struct {
	songSelectionMode bool
	ui                *UI
	playback          *PlaybackManager
	uiChan            chan UIRequest
	cmdChan           chan Command
	quitChan          chan struct{}
	quitOnce          sync.Once
}

func NewCommandHandler(playback *PlaybackManager, ui *UI) *CommandHandler {
	return &CommandHandler{
		songSelectionMode: false,
		playback:          playback,
		cmdChan:           make(chan Command),
		quitChan:          playback.quitChan,
		ui:                ui,
	}
}

func (h *CommandHandler) handleKeyPresses() {
	for kp := range h.ui.keyPressChan {
		if h.songSelectionMode {
			h.ui.songSelectionChan <- kp
		} else {
			h.cmdChan <- translateKeyPress(kp)
		}
	}
}

func (h *CommandHandler) handleCommands() {
	for cmd := range h.cmdChan {
		h.Handle(cmd)
	}
}

func (h *CommandHandler) Start() {
	go h.handleCommands()
	go h.handleKeyPresses()
}

func (h *CommandHandler) Handle(cmd Command) {
	switch cmd.Kind {
	case CmdAddToQueue:
		h.handleAddToQueue()
	case CmdSkip:
		h.handleSkip()
	case CmdShowQueue:
		fmt.Println(h.playback.playlist)
	case CmdSeek:
		h.handleSeek(cmd)
	case CmdTogglePause:
		h.HandleTogglePause()
	case CmdChangeSpeed:
		h.handleChangeSpeed(cmd)
	case CmdChangeVolume:
		h.handleChangeVolume(cmd)
	case CmdQuit:
		h.handleQuit()
	}
}

func (h *CommandHandler) requestSongSelection() Song {
	return h.ui.selectSong()
}

func (h *CommandHandler) handleSkip() {
	h.playback.player.Skip()
}

func (h *CommandHandler) HandleTogglePause() {
	h.playback.player.TogglePause()
	h.ui.updatePlaybackBox()
}

func (h *CommandHandler) handleAddToQueue() {
	h.songSelectionMode = true

	h.ui.uiRequestChan <- UISelectSong
	queueSong := <-h.ui.selectionResultChan

	h.songSelectionMode = false

	if queueSong != (Song{}) {
		(*h).playback.playlist = append(h.playback.playlist, queueSong)
		h.playback.notifyNextSong()
	}
}

func (h *CommandHandler) handleSeek(cmd Command) {
	if delta, ok := cmd.Value.(time.Duration); h.playback.player.Streamer != nil && ok {
		h.playback.player.Seek(delta)
		h.ui.updatePlaybackBox()
	} else {
		fmt.Println("Invalid type for time duration")
	}
}

func (h *CommandHandler) handleChangeSpeed(cmd Command) {
	if delta, ok := cmd.Value.(float64); ok {
		h.playback.player.ChangeSpeed(delta)
		h.ui.updatePlaybackBox()
	} else {
		fmt.Println("Invalid type for changing speed")
	}
}

func (h *CommandHandler) handleChangeVolume(cmd Command) {
	if delta, ok := cmd.Value.(float64); ok {
		h.playback.player.ChangeVolume(delta)
		h.ui.updatePlaybackBox()
	} else {
		fmt.Println("Invalid type for changing volume")
	}
}

func (h *CommandHandler) handleQuit() {
	h.quitOnce.Do(func() {
		select {
		case h.quitChan <- struct{}{}:
		default:
		}
	})
}
