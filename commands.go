package main

import (
	"github.com/eiannone/keyboard"
	"fmt"
	"time"
	"os"
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
		return Command{Kind: CmdSeek, Value: -seekStep}
	case 'm', 'M':
		return Command{Kind: CmdSeek, Value: seekStep}
	default:
		return Command{Kind: CmdNone}
	}
}

func translateKey(k keyboard.Key) Command {
	switch k {
	case keyboard.KeySpace:
		return Command{Kind: CmdTogglePause}
	case keyboard.KeyArrowLeft:
		return Command{Kind: CmdChangeSpeed, Value: -speedStep}
	case keyboard.KeyArrowRight:
		return Command{Kind: CmdChangeSpeed, Value: speedStep}
	case keyboard.KeyArrowDown:
		return Command{Kind: CmdChangeVolume, Value: -volumeStep}
	case keyboard.KeyArrowUp:
		return Command{Kind: CmdChangeVolume, Value: volumeStep}
	case keyboard.KeyEsc:
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
	songSelectionMode *bool
	songSelectionChan chan KeyPress
	allSongs          []Song
	player            *Player
	hasSongs          chan struct{}
	showQueueChan     chan struct{}
	quitChan          chan struct{}
}

func NewCommandHandler(
	songSelectionMode *bool,
	songSelectionChan chan KeyPress,
	allSongs []Song,
	player *Player,
	hasSongs chan struct{},
	showQueueChan chan struct{},
	quitChan chan struct{},
) *CommandHandler {
	return &CommandHandler{
		songSelectionMode: songSelectionMode,
		songSelectionChan: songSelectionChan,
		allSongs:          allSongs,
		player:            player,
		hasSongs:          hasSongs,
		showQueueChan:     showQueueChan,
		quitChan:          quitChan,
	}
}

func (h *CommandHandler) Start(cmdChan chan Command) {
	go func() {
		for cmd := range cmdChan {
			h.Handle(cmd)
		}
	}()
}

func (h *CommandHandler) Handle(cmd Command) {
	switch cmd.Kind {
	case CmdAddToQueue:
		h.handleAddToQueue()
	case CmdSkip:
		h.player.Skip()
	case CmdShowQueue:
		h.showQueueChan <- struct{}{}
	case CmdSeek:
		h.handleSeek(cmd)
	case CmdTogglePause:
		h.player.TogglePause()
	case CmdChangeSpeed:
		h.handleChangeSpeed(cmd)
	case CmdChangeVolume:
		h.handleChangeVolume(cmd)
	case CmdQuit:
		h.handleQuit()
	}
}

func (h *CommandHandler) handleAddToQueue() {
	listSongs(h.allSongs)
	*h.songSelectionMode = true
	queueSong := chooseSong(h.songSelectionChan, h.allSongs)
	*h.songSelectionMode = false
	fmt.Println(queueSong.Name)
	(*h).player.Playlist = append(h.player.Playlist, queueSong)
	select {
	case h.hasSongs <- struct{}{}:
	default:
	}
}

func (h *CommandHandler) handleSeek(cmd Command) {
	if delta, ok := cmd.Value.(time.Duration); h.player.Streamer != nil && ok {
		h.player.Seek(delta)
	} else {
		fmt.Println("Invalid type for time duration")
	}
}

func (h *CommandHandler) handleChangeSpeed(cmd Command) {
	if delta, ok := cmd.Value.(float64); ok {
		h.player.ChangeSpeed(delta)
	} else {
		fmt.Println("Invalid type for changing speed")
	}
}

func (h *CommandHandler) handleChangeVolume(cmd Command) {
	if delta, ok := cmd.Value.(float64); ok {
		h.player.ChangeVolume(delta)
	} else {
		fmt.Println("Invalid type for changing volume")
	}
}

func (h *CommandHandler) handleQuit() {
	select {
	case h.quitChan <- struct{}{}:
	default:
		typeMessage("Thhhhanks for using Axiom MP3-player", 0)
		h.player.updateTick.Stop()
		time.Sleep(time.Second)
		keyboard.Close()
		os.Exit(0)
	}
}
