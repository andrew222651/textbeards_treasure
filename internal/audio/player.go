package audio

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	cannonFireAsset       = "assets/cannon_fire.ogg"
	defaultMusicAsset     = "assets/default_music.wav"
	tavernMusicAsset      = "assets/tavern.ogg"
	musicCrossfadeOverlap = 500 * time.Millisecond
)

//go:embed assets/cannon_fire.ogg assets/default_music.wav assets/tavern.ogg
var embeddedAssets embed.FS

type playerCommand struct {
	name           string
	args           []string
	seekBeforePath func(time.Duration) []string
	seekAfterPath  func(time.Duration) []string
}

func (c playerCommand) argsFor(path string, offset time.Duration) []string {
	args := append([]string{}, c.args...)
	if offset > 0 && c.seekBeforePath != nil {
		args = append(args, c.seekBeforePath(offset)...)
	}
	args = append(args, path)
	if offset > 0 && c.seekAfterPath != nil {
		args = append(args, c.seekAfterPath(offset)...)
	}
	return args
}

// CannonFirePlayer plays the embedded cannon sound through the first available
// local audio command. Missing audio tools are treated as a silent no-op.
type CannonFirePlayer struct {
	prepareOnce sync.Once
	commandOnce sync.Once
	path        string
	prepareErr  error
	command     *playerCommand
}

func NewCannonFirePlayer() *CannonFirePlayer {
	return &CannonFirePlayer{}
}

func (p *CannonFirePlayer) Play() {
	if p == nil {
		return
	}
	path, ok := p.soundPath()
	if !ok {
		return
	}
	command, ok := p.playCommand()
	if !ok {
		return
	}

	cmd := exec.Command(command.name, command.argsFor(path, 0)...)
	if err := cmd.Start(); err != nil {
		return
	}
	go func() { _ = cmd.Wait() }()
}

func (p *CannonFirePlayer) Close() error {
	if p == nil || p.path == "" {
		return nil
	}
	return os.Remove(p.path)
}

func (p *CannonFirePlayer) soundPath() (string, bool) {
	p.prepareOnce.Do(func() {
		p.path, p.prepareErr = writeEmbeddedAsset(cannonFireAsset, "pirates-cannon-fire-*.ogg")
	})
	return p.path, p.prepareErr == nil && p.path != ""
}

func (p *CannonFirePlayer) playCommand() (playerCommand, bool) {
	p.commandOnce.Do(func() {
		for _, command := range cannonFireCommands() {
			if _, err := exec.LookPath(command.name); err == nil {
				selected := command
				p.command = &selected
				return
			}
		}
	})
	if p.command == nil {
		return playerCommand{}, false
	}
	return *p.command, true
}

type musicTrack int

const (
	defaultMusicTrack musicTrack = iota
	tavernMusicTrack
)

type musicAsset struct {
	pathPattern string
	assetPath   string
}

type musicLoop struct {
	track  musicTrack
	cancel context.CancelFunc
	done   chan struct{}
}

// MusicPlayer loops embedded music until stopped. Missing seek-capable audio
// tools are treated as a silent no-op.
type MusicPlayer struct {
	mu          sync.Mutex
	commandOnce sync.Once
	stopOnce    sync.Once
	paths       map[musicTrack]string
	prepareErrs map[musicTrack]error
	offsets     map[musicTrack]time.Duration
	command     *playerCommand
	current     *musicLoop
	active      map[*musicLoop]struct{}
	stopped     bool
}

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{}
}

func (p *MusicPlayer) Start() {
	p.switchTo(defaultMusicTrack)
}

func (p *MusicPlayer) SetInPort(inPort bool) {
	if inPort {
		p.EnterPort()
		return
	}
	p.LeavePort()
}

func (p *MusicPlayer) EnterPort() {
	p.switchTo(tavernMusicTrack)
}

func (p *MusicPlayer) LeavePort() {
	p.switchTo(defaultMusicTrack)
}

func (p *MusicPlayer) Stop() {
	if p == nil {
		return
	}
	p.stopOnce.Do(func() {
		p.mu.Lock()
		p.stopped = true
		loops := make([]*musicLoop, 0, len(p.active))
		for loop := range p.active {
			loops = append(loops, loop)
		}
		paths := make([]string, 0, len(p.paths))
		for _, path := range p.paths {
			paths = append(paths, path)
		}
		p.current = nil
		p.mu.Unlock()

		for _, loop := range loops {
			stopMusicLoop(loop)
		}
		for _, path := range paths {
			_ = os.Remove(path)
		}
	})
}

func (p *MusicPlayer) switchTo(track musicTrack) {
	if p == nil {
		return
	}
	p.mu.Lock()
	stopped := p.stopped
	p.mu.Unlock()
	if stopped {
		return
	}
	path, ok := p.musicPath(track)
	if !ok {
		return
	}
	command, ok := p.playCommand()
	if !ok {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	loop := &musicLoop{track: track, cancel: cancel, done: make(chan struct{})}

	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		cancel()
		return
	}
	if p.current != nil && p.current.track == track {
		p.mu.Unlock()
		cancel()
		return
	}
	if p.active == nil {
		p.active = make(map[*musicLoop]struct{})
	}
	old := p.current
	p.current = loop
	p.active[loop] = struct{}{}
	p.mu.Unlock()

	go p.loop(ctx, loop, command, path)
	if old != nil {
		go stopMusicLoopAfter(old, musicCrossfadeOverlap)
	}
}

func (p *MusicPlayer) loop(ctx context.Context, loop *musicLoop, command playerCommand, path string) {
	defer func() {
		p.mu.Lock()
		delete(p.active, loop)
		p.mu.Unlock()
		close(loop.done)
	}()

	for ctx.Err() == nil {
		offset := p.trackOffset(loop.track)
		startedAt := time.Now()
		cmd := exec.CommandContext(ctx, command.name, command.argsFor(path, offset)...)
		err := cmd.Run()

		if ctx.Err() != nil {
			p.setTrackOffset(loop.track, offset+time.Since(startedAt))
			return
		}
		if err == nil {
			p.setTrackOffset(loop.track, 0)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (p *MusicPlayer) trackOffset(track musicTrack) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.offsets[track]
}

func (p *MusicPlayer) setTrackOffset(track musicTrack, offset time.Duration) {
	if offset < 0 {
		offset = 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.offsets == nil {
		p.offsets = make(map[musicTrack]time.Duration)
	}
	p.offsets[track] = offset
}

func stopMusicLoopAfter(loop *musicLoop, delay time.Duration) {
	if loop == nil {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-loop.done:
		return
	case <-timer.C:
		stopMusicLoop(loop)
	}
}

func stopMusicLoop(loop *musicLoop) {
	if loop == nil {
		return
	}
	loop.cancel()
	<-loop.done
}

func (p *MusicPlayer) musicPath(track musicTrack) (string, bool) {
	asset, ok := musicAssets()[track]
	if !ok {
		return "", false
	}

	p.mu.Lock()
	if p.paths == nil {
		p.paths = make(map[musicTrack]string)
	}
	if p.prepareErrs == nil {
		p.prepareErrs = make(map[musicTrack]error)
	}
	if path := p.paths[track]; path != "" {
		ok := p.prepareErrs[track] == nil
		p.mu.Unlock()
		return path, ok
	}
	p.mu.Unlock()

	path, err := writeEmbeddedAsset(asset.assetPath, asset.pathPattern)

	p.mu.Lock()
	defer p.mu.Unlock()
	if existing := p.paths[track]; existing != "" {
		if path != "" {
			_ = os.Remove(path)
		}
		return existing, p.prepareErrs[track] == nil
	}
	p.paths[track] = path
	p.prepareErrs[track] = err
	return path, err == nil && path != ""
}

func (p *MusicPlayer) playCommand() (playerCommand, bool) {
	p.commandOnce.Do(func() {
		for _, command := range musicCommands() {
			if _, err := exec.LookPath(command.name); err == nil {
				selected := command
				p.command = &selected
				return
			}
		}
	})
	if p.command == nil {
		return playerCommand{}, false
	}
	return *p.command, true
}

func writeEmbeddedAsset(assetPath, pattern string) (string, error) {
	data, err := embeddedAssets.ReadFile(assetPath)
	if err != nil {
		return "", err
	}

	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func musicAssets() map[musicTrack]musicAsset {
	return map[musicTrack]musicAsset{
		defaultMusicTrack: {assetPath: defaultMusicAsset, pathPattern: "pirates-default-music-*.wav"},
		tavernMusicTrack:  {assetPath: tavernMusicAsset, pathPattern: "pirates-tavern-music-*.ogg"},
	}
}

func cannonFireCommands() []playerCommand {
	return []playerCommand{
		{name: "ffplay", args: []string{"-nodisp", "-autoexit", "-loglevel", "quiet"}},
		{name: "mpv", args: []string{"--no-terminal", "--really-quiet"}},
		{name: "pw-play"},
		{name: "paplay"},
		{name: "ogg123", args: []string{"-q"}},
		{name: "play", args: []string{"-q"}},
		{name: "canberra-gtk-play", args: []string{"-f"}},
	}
}

func musicCommands() []playerCommand {
	return []playerCommand{
		{name: "ffplay", args: []string{"-nodisp", "-autoexit", "-loglevel", "quiet"}, seekBeforePath: ffplaySeekArgs},
		{name: "mpv", args: []string{"--no-terminal", "--really-quiet"}, seekBeforePath: mpvSeekArgs},
		{name: "play", args: []string{"-q"}, seekAfterPath: soxPlaySeekArgs},
	}
}

func ffplaySeekArgs(offset time.Duration) []string {
	return []string{"-ss", seekSeconds(offset)}
}

func mpvSeekArgs(offset time.Duration) []string {
	return []string{"--start=" + seekSeconds(offset)}
}

func soxPlaySeekArgs(offset time.Duration) []string {
	return []string{"trim", seekSeconds(offset)}
}

func seekSeconds(offset time.Duration) string {
	if offset < 0 {
		offset = 0
	}
	return fmt.Sprintf("%.3f", offset.Seconds())
}
