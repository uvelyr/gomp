package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"github.com/gdamore/tcell/v2"
)

var ErrNoSongs = errors.New("No songs in playlist")

func FindSongs(path string) ([]Song, error) {
	var songs []Song
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			nestedSongs, err := FindSongs(fullPath)
			if err != nil {
				return nil, err
			}
			songs = append(songs, nestedSongs...)
		} else if filepath.Ext(entry.Name()) == ".mp3" {
			songs = append(songs, Song{
				Path: filepath.Join(path, entry.Name()),
				Name: entry.Name(),
			})
		}
	}
	return songs, nil
}

func setupChannels() (quitChan chan struct{}, updateTimer chan struct{}) {
	quitChan = make(chan struct{})
	updateTimer = make(chan struct{})
	return quitChan, updateTimer
}

func scanSongs() []Song {
	allSongs, err := FindSongs(".")
	if err != nil {
		log.Fatal(err)
	}
	return allSongs
}

func main() {
	//Greet()

	// scan for songs
	allSongs := scanSongs()

	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}

	if err := screen.Init(); err != nil {
		log.Fatal(err)
	}

	defer screen.Fini()
	defer screen.Clear()

	// create player
	player := NewPlayer()

	ui := NewUI(screen, &PlaybackManager{}, allSongs)
	uiChan := ui.uiRequestChan

	// Create playback manager
	playbackManager := NewPlaybackManager(player, uiChan)
	ui.playback = playbackManager

	// Create command handler
	cmdHandler := NewCommandHandler(playbackManager, ui)

	ui.start()

	cmdHandler.Start()

	playbackManager.Start()
}
