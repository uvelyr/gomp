package main

import (
	"log"
	"math/rand"
	"strconv"
	//	"io/fs"
	"fmt"
	"os"
	"path/filepath"
	"time"
	//	"strings"
	"github.com/eiannone/keyboard"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
)

const (
	speedStep  = 0.05
	volumeStep = 0.35
	seekStep = time.Second * 5
	minVolume = -5.0
	maxVolume = 2.0
	minSpeed = 0.6
	maxSpeed = 2.0
)


type Song struct {
	Path string //path to a file, including the file itself
	Name string //name of a fle
}

func listSongs(songs []Song) {
	for i, song := range songs {
		fmt.Println(strconv.Itoa(i+1) + ": " + song.Name)
	}
}

func chooseSong(songs []Song) Song {
	fmt.Println("Enter the number of song: ")

	var input []rune

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			log.Fatal(err)
		}

		switch key {
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
			typeMessage("Thanks for using Axiom MP3-player", 0)
			time.Sleep(time.Second)
			os.Exit(0)
		default:
			if char >= '0' && char <= '9' {
				input = append(input, char)
				fmt.Printf("%c", char)
			}
		}
	}
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo { return lo }
	if v > hi {
		fmt.Println("RETURNING MAX VALUE")
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo { return lo }
	if v > hi { return hi }
	return v
}

func changeVolume(v *effects.Volume, delta float64) {
	newVolume := clampFloat(v.Volume+delta, minVolume, maxVolume)

	speaker.Lock()
	v.Volume = newVolume

	if v.Volume <= minVolume {
		v.Silent = true
	} else {
		v.Silent = false
	}
	speaker.Unlock()
}

func changeSpeed(r *beep.Resampler, delta float64) {
	newRatio := clampFloat(r.Ratio()+delta, minSpeed, maxSpeed)
	speaker.Lock()
	r.SetRatio(newRatio)
	speaker.Unlock()
}

func changeTime(streamer beep.StreamSeekCloser, format beep.Format, delta time.Duration) {
	speaker.Lock()
	cur := streamer.Position()
	deltaSamples := format.SampleRate.N(delta)
	newPos := clampInt(cur+deltaSamples, 0, streamer.Len())
	streamer.Seek(newPos)
	speaker.Unlock()
}

func togglePause(ctrl *beep.Ctrl) {
	speaker.Lock()
	ctrl.Paused = !ctrl.Paused
	speaker.Unlock()
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

func main() {
	typeMessage("Welcome to Axiom MP3-player", 0)

	//declare song controllers
	var streamer beep.StreamSeekCloser
	var format beep.Format
	var control *beep.Ctrl
	var speed *beep.Resampler
	var volume *effects.Volume

	var playlist []Song

	//declare signal channels
	quitChan := make(chan struct{})
	songDoneChan := make(chan struct{})
	songSkipChan := make(chan struct{})
	showPlaylistChan := make(chan struct{})
	updateTimer := make(chan struct{})
	hasSongs := make(chan struct{}, 1)

	//scan for songs
	allSongs, err := FindSongs(".")
	if err != nil {
		log.Fatal(err)
	}

	//print control bindings
	fmt.Println("Up and Down keys — change volume")
	fmt.Println("Left and Right keys — change speed")
	fmt.Println("Enter — toggle pause")
	fmt.Println("Q — add song to query")
	fmt.Println("S - skip song")
	fmt.Println("D - show query")

	//list songs with their indexes. UI
	listSongs(allSongs)

	//enable real-time input mode
	err = keyboard.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer keyboard.Close()

	go func() {
		for {
			char, key, err := keyboard.GetKey()
			if err != nil {
				log.Fatal(err)
			}

			switch char {
			case 'q', 'Q':
				listSongs(allSongs)
				queueSong := chooseSong(allSongs)
				fmt.Println(queueSong.Name)
				playlist = append(playlist, queueSong)
				select { case hasSongs <- struct{}{}: default: }
				continue
			case 's', 'S':
				if control != nil {
					songSkipChan <- struct{}{}
				}
			case 'd', 'D':
				showPlaylistChan <- struct{}{}
			case 'n', 'N':
				if streamer != nil {
					changeTime(streamer, format, -seekStep)
				}
			case 'm', 'M':
				if streamer != nil {
					changeTime(streamer, format, seekStep)
				}
			}

			if control != nil && volume != nil && speed != nil {
				switch key {
				case keyboard.KeyEnter:
					togglePause(control)
				case keyboard.KeyArrowLeft:
					changeSpeed(speed, -speedStep)
				case keyboard.KeyArrowRight:
					changeSpeed(speed, speedStep)
				case keyboard.KeyArrowUp:
					changeVolume(volume, volumeStep)
				case keyboard.KeyArrowDown:
					changeVolume(volume, -volumeStep)
				case keyboard.KeyEsc:
					select {
					case quitChan <- struct{}{}:
					default:
						typeMessage("Thanks for using Axiom MP3-player", 0)
						time.Sleep(time.Second)
						keyboard.Close()
						os.Exit(0)
					}
				}
			}
		}
	}()


	go func() {
		for {
			<- showPlaylistChan
			fmt.Println(playlist)
		}
	}()

	for {
		//wait until there is a signal that there are songs
		<- hasSongs

		if len(playlist) == 0 {
			continue
		}

		file, err := os.Open(playlist[0].Path)
		if err != nil {
			log.Fatal(err)
		}
		streamer, format, err = mp3.Decode(file)
		if err != nil {
			log.Fatal(err)
		}
		//setting up a speaker
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

		//initiating a streamer.
		control = &beep.Ctrl{Streamer: streamer, Paused: false}
		volume = &effects.Volume{
			Streamer: control,
			Volume:   0,
			Silent:   false,
			Base:   2,
		}
		speed = beep.ResampleRatio(4, 1, volume)

		//play song
		speaker.Play(beep.Seq(speed, beep.Callback(func() {
			fmt.Println("SONG DONE, SENDING SIGNAL")
			songDoneChan <- struct{}{}
			fmt.Println("SIGNAL SENT")
		})))

		//delete played song from playlist
		playlist = playlist[1:]

		//declare a ticker
		ticker := time.NewTicker(time.Second)

		innerloop:
		for {
			//wait till one of the actions is songDoneChan
			select {
			case <- songSkipChan:
				speaker.Lock()
				control.Paused = true
				speaker.Unlock()

				//if there are any remaining songs in the playlist, send signal to hasSongs
				//to unblock player
				if len(playlist) > 0 {
					//will keep for hasSongs to start listening
					select {
					case hasSongs <- struct{}{}:
					default:
					}
				}
				break innerloop
			case <- songDoneChan:
				//if there are any remaining songs in the playlist, send signal to hasSongs
				//to unblock player
				fmt.Println("Song is done!")
				if len(playlist) > 0 {
					//will keep for hasSongs to start listening
					select {
					case hasSongs <- struct{}{}:
					default:
					}
				}
				break innerloop
			case <- ticker.C:
				updateTime(streamer, format)
			case <- updateTimer:
				updateTime(streamer, format)
			case <- quitChan:
				typeMessage("Thanks for using Axiom MP3-player", 0)
				time.Sleep(time.Second)
				return
			}
		}
		//stop the ticker
		ticker.Stop()

		//close streamer and file
		streamer.Close()
		file.Close()
	}
}
