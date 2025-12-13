package main

import (
	"fmt"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"os"
	"time"
)

type Player struct {
	Streamer beep.StreamSeekCloser
	File     *os.File
	Format   beep.Format
	Control  *beep.Ctrl
	Speed    *beep.Resampler
	Volume   *effects.Volume

	//internal channgels for controlling
	songDone   chan struct{}
	songSkip   chan struct{}
	updateTick *time.Ticker
}

func NewPlayer() *Player {
	return &Player{
		songDone:   make(chan struct{}),
		songSkip:   make(chan struct{}),
		updateTick: time.NewTicker(time.Second),
	}
}

func (pm *PlaybackManager) PlayNext() error {
	p := pm.player
	if len(pm.playlist) == 0 {
		return ErrNoSongs
	}

	file, err := os.Open(pm.playlist[0].Path)
	if err != nil {
		return fmt.Errorf("Couldn't open file: %w", err)
	}

	streamer, format, err := mp3.Decode(file)
	if err != nil {
		file.Close()
		return fmt.Errorf("Couldn't decode file: %w", err)
	}

	p.File = file
	p.Streamer = streamer
	p.Format = format

	//initiating a streamer.
	p.Control = &beep.Ctrl{Streamer: streamer, Paused: false}
	p.Volume = &effects.Volume{
		Streamer: p.Control,
		Volume:   0,
		Silent:   false,
		Base:     2,
	}
	p.Speed = beep.ResampleRatio(4, 1, p.Volume)

	//setting up a speaker
	speaker.Init(format.SampleRate, format.SampleRate.N(bufferSize))

	//play song
	speaker.Play(beep.Seq(p.Speed, beep.Callback(func() {
		p.songDone <- struct{}{}
	})))

	pm.currentSong = &Song{
		Path: pm.playlist[0].Path,
		Name: pm.playlist[0].Name,
	}
	pm.playlist = pm.playlist[1:]
	return nil
}

func (p *Player) TogglePause() error {
	speaker.Lock()
	p.Control.Paused = !p.Control.Paused
	speaker.Unlock()
	return nil
}

func (p *Player) ChangeVolume(delta float64) error {
	newVolume := clampFloat(p.Volume.Volume+delta, minVolume, maxVolume)
	speaker.Lock()
	p.Volume.Volume = newVolume
	p.Volume.Silent = p.Volume.Volume <= minVolume
	speaker.Unlock()
	return nil
}

func (p *Player) ChangeSpeed(delta float64) error {
	newRatio := clampFloat(p.Speed.Ratio()+delta, minSpeed, maxSpeed)
	speaker.Lock()
	p.Speed.SetRatio(newRatio)
	speaker.Unlock()
	return nil
}

func (p *Player) Seek(delta time.Duration) error {
	speaker.Lock()
	curPos := p.Streamer.Position()
	deltaSamples := p.Format.SampleRate.N(delta)
	newPos := clampInt(curPos+deltaSamples, 0, p.Streamer.Len())
	p.Streamer.Seek(newPos)
	speaker.Unlock()
	return nil
}

func (p *Player) Skip() error {
	p.songSkip <- struct{}{}
	return nil
}

type PlaybackManager struct {
	player            *Player
	currentSong       *Song
	playlist          []Song
	uiChan            chan<- UIRequest
	hasSongs          chan struct{}
	quitChan          chan struct{}
	updateTimer       chan struct{}
}

func NewPlaybackManager(player *Player, uiChan chan<- UIRequest) *PlaybackManager {
	return &PlaybackManager{
		player:      player,
		currentSong: nil,
		playlist:    []Song{},
		uiChan:      uiChan,
		hasSongs:    make(chan struct{}, 1),
		quitChan:    make(chan struct{}),
		updateTimer: make(chan struct{}),
	}
}

func (pm *PlaybackManager) Start() {
	for {
		// Wait until there is a signal that there are songs
		select {
		case <-pm.hasSongs:
			pm.PlayNext()

			if !pm.handlePlayback() {
				return
			}

			pm.cleanupCurrentSong()
		case <-pm.quitChan:
			pm.handleQuit()
			return
		}
	}
}

func (pm *PlaybackManager) handlePlayback() bool {
	for {
		select {
		case <-pm.player.songSkip:
			if len(pm.playlist) > 0 {
				return pm.handleSongSkip()
			}
		case <-pm.player.songDone:
			return pm.handleSongDone()
		case <-pm.player.updateTick.C:
			pm.uiChan <- UIUpdatePlaybackBox
		case <-pm.updateTimer:
			pm.uiChan <- UIUpdatePlaybackBox
		case <-pm.quitChan:
			pm.handleQuit()
			return false
		}
	}
}

func (pm *PlaybackManager) handleSongSkip() bool {
	pm.player.TogglePause()
	pm.notifyNextSong()
	pm.uiChan <- UIUpdatePlaybackBox
	return true
}

func (pm *PlaybackManager) handleSongDone() bool {
	pm.currentSong = nil
	pm.uiChan <- UIUpdatePlaybackBox
	pm.notifyNextSong()
	return true
}

func (pm *PlaybackManager) notifyNextSong() {
	if len(pm.playlist) > 0 {
		select {
		case pm.hasSongs <- struct{}{}:
		default:
		}
	}
}

func (pm *PlaybackManager) handleQuit() {
	fmt.Println("Thanks for using Axiom MP3-player")
	pm.player.updateTick.Stop()

	if pm.player.Streamer != nil {
		pm.player.Streamer.Close()
	}

	close(pm.player.songSkip)
	close(pm.hasSongs)

	time.Sleep(time.Millisecond * 500)
}

func (pm *PlaybackManager) cleanupCurrentSong() {
	//	if pm.player.Streamer != nil {
	//		if err := pm.player.Streamer.Close(); err != nil {
	//			fmt.Println("Error closing streamer: ", err)
	//		}
	//	}
	//
	// pm.player.Streamer = nil
	// pm.player.File = nil
}
