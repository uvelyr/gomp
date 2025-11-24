package main

import (
	"fmt"
	"os"
	"time"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
)

func NewPlayer() *Player {
	return &Player{
		Playlist: []Song{},
		songDone: make(chan struct{}),
		songSkip: make(chan struct{}),
		updateTick: time.NewTicker(time.Second),
	}
}

func (p *Player) PlayNext() error {
	if len(p.Playlist) == 0 {
		return ErrNoSongs
	}

	file, err := os.Open(p.Playlist[0].Path)
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
		Base:   2,
	}
	p.Speed = beep.ResampleRatio(4, 1, p.Volume)

	//setting up a speaker
	speaker.Init(format.SampleRate, format.SampleRate.N(bufferSize))

	//play song
	speaker.Play(beep.Seq(p.Speed, beep.Callback(func() {
		p.songDone <- struct{}{}
	})))

	//delete played song from playlist
	p.Playlist = p.Playlist[1:]
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
	newPos := clampInt(curPos + deltaSamples, 0, p.Streamer.Len())
	p.Streamer.Seek(newPos)
	speaker.Unlock()
	return nil
}

func (p *Player) Skip() error {
	p.songSkip <- struct{}{}
	return nil
}
