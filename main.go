package main

import (
	"log"
	"math/rand"
	"strconv"
	"errors"
	//	"io/fs"
	"fmt"
	"os"
	"path/filepath"
	"time"
	//	"strings"
	"github.com/eiannone/keyboard"
	"github.com/gopxl/beep"
)

const (
	bufferSize = time.Second / 10
	speedStep  = 0.05
	volumeStep = 0.35
	seekStep = time.Second * 5
	minVolume = -5.0
	maxVolume = 2.0
	minSpeed = 0.6
	maxSpeed = 2.0
)

var ErrNoSongs = errors.New("No songs in playlist")

func listSongs(songs []Song) {
	for i, song := range songs {
		fmt.Println(strconv.Itoa(i+1) + ": " + song.Name)
	}
}

func chooseSong(h *CommandHandler) Song {
	eventChan := h.songSelectionChan
	songs := h.allSongs

	fmt.Println("Enter the number of song: ")

	var input []rune

	for ev := range eventChan{
		switch ev.Key {
		case keyboard.KeyEnter:
			if len(input) == 0 {
				fmt.Println("Invalid input, try again: ")
				continue
			}

			fmt.Println()
			choiceStr := string(input)
			choice, err := strconv.Atoi(choiceStr)
			if err != nil || choice < 1 || choice > len(songs) {
				fmt.Println("Invalid number, try again:")
				input = nil
				continue
			}
			return songs[choice-1]
		case keyboard.KeyBackspace, keyboard.KeyBackspace2:
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b")
			}
		case keyboard.KeyEsc:
			fmt.Println("h")
			h.handleQuit()
			return Song{}
		default:
			if ev.Rune >= '0' && ev.Rune <= '9' {
				input = append(input, ev.Rune)
				fmt.Printf("%c", ev.Rune)
			}
		}
	}
	return Song{}
}

func typeMessage(text string, baseDelay time.Duration) {
	for _, char := range text {
		fmt.Printf("%c", char)
		randomDelay := baseDelay + time.Duration(rand.Intn(200))*time.Millisecond
		time.Sleep(randomDelay)

	}
	fmt.Println()
}

func updateTime(streamer beep.StreamSeekCloser, format beep.Format) {
	elapsed := format.SampleRate.D(streamer.Position()).Round(time.Second)
	total := format.SampleRate.D(streamer.Len()).Round(time.Second)
	fmt.Printf("%v / %v\n", elapsed, total)
}

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



func setupChannels() (
	quitChan chan struct{},
	showQueueChan chan struct{},
	updateTimer chan struct{},
	hasSongs chan struct{},
	cmdChan chan Command,
	songSelectionChan chan KeyPress,
	keyPressChan chan KeyPress,
) {
	quitChan = make(chan struct{})
	showQueueChan = make(chan struct{})
	updateTimer = make(chan struct{})
	hasSongs = make(chan struct{}, 1)
	cmdChan = make(chan Command)
	songSelectionChan = make(chan KeyPress, 16)
	keyPressChan = make(chan KeyPress, 16)

	return quitChan, showQueueChan, updateTimer, hasSongs, cmdChan, songSelectionChan, keyPressChan
}

func Greet() {
	typeMessage("Welcome to Axiom MP3-player", 0)
}

func scanSongs() []Song {
	allSongs, err := FindSongs(".")
	if err != nil {
		log.Fatal(err)
	}
	return allSongs
}

func printControls() {
	fmt.Println("Up and Down keys — change volume")
	fmt.Println("Left and Right keys — change speed")
	fmt.Println("Enter — toggle pause")
	fmt.Println("Q — add song to query")
	fmt.Println("S - skip song")
	fmt.Println("D - show query")
}

func initializeKeyboard() error {
	err := keyboard.Open()
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func startRawInputRoutine(keyPressChan chan KeyPress) {
	go func() {
		for {
			//read a key from keyboard
			char, key, err := keyboard.GetKey()
			if err != nil {
				fmt.Println("Couldn't get key:", err)
			}
			keyPressChan <- KeyPress{Rune: char, Key: key}
		}
	}()
}

func startPlayerInputRoutine(
	keyPressChan chan KeyPress,
	cmdChan chan Command,
	songSelectionChan chan KeyPress,
	songSelectionMode *bool,
) {
	go func() {
		for kp := range keyPressChan {
			if *songSelectionMode {
				songSelectionChan <- kp
			} else {
				cmdChan <- translateKeyPress(kp)
			}
		}
	}()
}

func main() {
	Greet()

	// scan for songs
	allSongs := scanSongs()

	printControls()
	listSongs(allSongs)

	initializeKeyboard()
	defer keyboard.Close()

	// set up channels
	quitChan, showQueueChan, updateTimer, hasSongs, cmdChan,
	 songSelectionChan, keyPressChan := setupChannels()

	songSelectionMode := false

	// create player
	player := NewPlayer()

	// launch input goroutines
	startRawInputRoutine(keyPressChan)
	startPlayerInputRoutine(keyPressChan, cmdChan, songSelectionChan, &songSelectionMode)

	// Create command handler
	cmdHandler := NewCommandHandler(
		&songSelectionMode,
		songSelectionChan,
		allSongs,
		player,
		hasSongs,
		showQueueChan,
		quitChan,
	)

	// Start command handler
	cmdHandler.Start(cmdChan)

	go func() {
		for range showQueueChan{
			fmt.Println(player.Playlist)
		}
	}()

	// Create and start playback manager
	playbackManager := NewPlaybackManager(player, hasSongs, quitChan, updateTimer)
	playbackManager.Start()
}
