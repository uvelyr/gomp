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
	bufferSize = time.Second / 10
	speedStep  = 0.05
	volumeStep = 0.35
	seekStep = time.Second * 5
	minVolume = -5.0
	maxVolume = 2.0
	minSpeed = 0.6
	maxSpeed = 2.0
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

type Command struct {
	Kind CommandKind
	Value any
}

type Song struct {
	Path string //path to a file, including the file itself
	Name string //name of a fle
}

type KeyPress struct {
	Rune rune
	Key keyboard.Key
}

type Player struct {
	Playlist []Song

	Streamer beep.StreamSeekCloser
	File *os.File
	Format beep.Format
	Control *beep.Ctrl
	Speed *beep.Resampler
	Volume *effects.Volume

	//internal channgels for controlling
	songDone chan struct{}
	songSkip chan struct{}
	updateTick *time.Ticker
}

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
		return nil
	}

	file, err := os.Open(p.Playlist[0].Path)
	if err != nil {
		fmt.Println("Couldn't open file: ", err)
		return err
	}

	streamer, format, err := mp3.Decode(file)
	if err != nil {
		fmt.Println("Couldn't decode file: ", err)
		return err
	}

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

func listSongs(songs []Song) {
	for i, song := range songs {
		fmt.Println(strconv.Itoa(i+1) + ": " + song.Name)
	}
}

func chooseSong(eventChan <-chan KeyPress, songs []Song) Song {
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
			typeMessage("Thanks for using Axiom MP3-player", 0)
			time.Sleep(time.Second)
			keyboard.Close()
			os.Exit(0)
		default:
			if ev.Rune >= '0' && ev.Rune <= '9' {
				input = append(input, ev.Rune)
				fmt.Printf("%c", ev.Rune)
			}
		}
	}
	return Song{}
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

func main() {
	typeMessage("Welcome to Axiom MP3-player", 0)

	//declare signal channels
	quitChan := make(chan struct{})
	showQueueChan := make(chan struct{})
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

	cmdChan := make(chan Command)
	songSelectionChan := make(chan KeyPress)
	playerControlChan := make(chan KeyPress)
	activeInputChan := playerControlChan

	player := NewPlayer()
	//raw input goroutine
	go func() {
		for {
			//read a key from keyboard
			char, key, err := keyboard.GetKey()
			if err != nil {
				fmt.Println("Couldn't get key:", err)
			}
			//send wherever curInChan points to
			activeInputChan <- KeyPress{Rune: char, Key: key}
		}
	}()

	//handle playerInput
	go func() {
		for {
			keyPress := <- playerControlChan

			if keyPress.Rune != 0 {
				cmdChan <- translateRune(keyPress.Rune)
			} else {
				cmdChan <- translateKey(keyPress.Key)
			}
		}
	}()

	go func() {
		for {
			cmd := <- cmdChan

			switch cmd.Kind {
			case CmdAddToQueue:
				listSongs(allSongs)
				activeInputChan = songSelectionChan
				queueSong := chooseSong(songSelectionChan, allSongs)
				activeInputChan = playerControlChan
				fmt.Println(queueSong.Name)
				player.Playlist = append(player.Playlist, queueSong)
				select { case hasSongs <- struct{}{}: default: }
			case CmdSkip:
				player.Skip()
			case CmdShowQueue:
				showQueueChan <- struct{}{}
			case CmdSeek:
				if delta, ok := cmd.Value.(time.Duration); player.Streamer != nil && ok {
					player.Seek(delta)
				} else {
					fmt.Println("Invalid type for time duration")
				}
			case CmdTogglePause:
				player.TogglePause()
			case CmdChangeSpeed:
				//type assertion
				if delta, ok := cmd.Value.(float64); ok {
					player.ChangeSpeed(delta)
				} else {
					fmt.Println("Invalid type for changing speed")
				}
			case CmdChangeVolume:
				//type assertion
				if delta, ok := cmd.Value.(float64); ok {
					player.ChangeVolume(delta)
				} else {
					fmt.Println("Invalid type for changing volume")
				}
			case CmdQuit:
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
	}()

	go func() {
		for {
			<- showQueueChan
			fmt.Println(player.Playlist)
		}
	}()

	for {
		//wait until there is a signal that there are songs
		<- hasSongs

		player.PlayNext()

		innerloop:
		for {
			//wait till one of the actions is done
			select {
			case <- player.songSkip:
				player.TogglePause()

				//if there are any remaining songs in the playlist, send signal to hasSongs
				//to unblock player
				if len(player.Playlist) > 0 {
					select {
					case hasSongs <- struct{}{}:
					default:
					}
				}
				break innerloop
			case <- player.songDone:
				//if there are any remaining songs in the playlist, send signal to hasSongs
				//to unblock player
				fmt.Println("Song is done!")
				if len(player.Playlist) > 0 {
					select {
					case hasSongs <- struct{}{}:
					default:
					}
				}
				break innerloop
			case <- player.updateTick.C:
				updateTime(player.Streamer, player.Format)
			case <- updateTimer:
				updateTime(player.Streamer, player.Format)
			case <- quitChan:
				typeMessage("Thanks for using Axiom MP3-player", 0)
				time.Sleep(time.Second)
				return
			}
		}
		//stop the ticker
		player.updateTick.Stop()

		//close streamer and file
		player.Streamer.Close()
		player.File.Close()
	}
}
