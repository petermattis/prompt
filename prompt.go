package prompt

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"unicode/utf8"

	"golang.org/x/term"
)

type state struct {
	history  history
	killRing killRing
	screen   screen

	// inputFinished is a callback invoked by the finish-or-enter command to
	// determine if the input is considered complete. If the callback is nil, or it
	// returns true, the input is considered complete and ReadLine will return the
	// input. Otherwise, a newline is inserted into the input. See the
	// WithInputFinished option for configuration.
	inputFinished func(text string) bool
}

// Prompt contains the state for reading single or multi-line input from a
// terminal. Similar to readline, libedit, and other CLI line reading libraries,
// Prompt provides support for basic editing functionality such as cursor
// movement, deletion, a kill ring, and history.
//
// Prompt supports a common subset of the universe of key input sequences which
// are used by ~75% of the terminals in the terminfo database, including most
// modern terminals. Prompt itself does not use terminfo. Additionally, Prompt
// requires that the terminal handle a minimal set of ANSI escape sequences for
// rendering text:
//
//   - cursor-up:           ESC[A
//   - cursor-down:         ESC[B
//   - cursor-right:        ESC[C
//   - cursor-left:         ESC[D
//   - cursor-home:         ESC[H
//   - erase-line-to-right: ESC[K
//   - erase-screen:        ESC[2J
//
// Prompt eschews using more advanced terminal operations such as insert/delete
// character and insert mode. This decision results in Prompt having to
// re-render more lines of text on editing operations, yet for line editing the
// difference usually amounts to sending a few hundred bytes to the terminal
// (for a long line). On modern hardware and networks, this amount of data is
// trivial. The benefit of eschewing more advanced terminal operations is that
// the same rendering output is used for all terminals as opposed to the
// libedit/readline approach which requires intimate knowledge of the terminal
// capabilities (via terminfo) and which can sometimes go horribly wrong
// resulting in corruption of the rendered text.
type Prompt struct {
	fd  int
	in  io.Reader
	out io.Writer

	// inBytes and inBuf are used by the reader loop to read data from the input.
	inBytes []byte
	inBuf   [256]byte
	prompt  []rune

	// bindings holds key bindings, mapping key input to an command to perform. If a
	// key is not present in the binding map it is inserted at the current cursor
	// position.
	bindings map[rune]command

	mu struct {
		sync.Mutex
		state state
	}
}

// New creates a new Prompt using the supplied options. If no options are
// specified, the Prompt uses os.Stdin and os.Stdout for input and output.
func New(options ...Option) *Prompt {
	p := &Prompt{
		in:       os.Stdin,
		out:      os.Stdout,
		bindings: make(map[rune]command),
	}

	if err := parseBindings(p.bindings, defaultBindings); err != nil {
		panic(err)
	}

	p.mu.state.screen.Init()
	for _, opt := range options {
		opt.apply(p)
	}

	type fdGetter interface {
		Fd() uintptr
	}
	if f, ok := p.in.(fdGetter); ok {
		p.fd = int(f.Fd())
	}
	return p
}

// Close closes the Prompt, releasing any open resources.
func (p *Prompt) Close() error {
	return nil
}

// ReadLine reads a line of input. If the input is canceled, io.EOF is returned
// as the error.
func (p *Prompt) ReadLine(prompt string) (string, error) {
	if err := p.updateSize(); err != nil {
		return "", err
	}

	if p.fd != -1 {
		// If we have a file descriptor, set up SIGWINCH handling so we can get notified
		// of changes in the terminal's size.
		winch := make(chan os.Signal, 1)
		signal.Notify(winch, syscall.SIGWINCH)
		go func() {
			for range winch {
				_ = p.updateSize()
			}
		}()
		defer func() {
			signal.Stop(winch)
			close(winch)
		}()

		// Put the terminal into raw mode, restoring the
		// original mode on exit.
		saved, err := term.MakeRaw(p.fd)
		if err != nil {
			return "", err
		}
		defer term.Restore(p.fd, saved)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.mu.state.screen.Reset([]rune(prompt))
	p.mu.state.screen.Flush(p.out)

	for {
		// Loop processing keys from the input.
		if result, err := p.processInputLocked(); err != nil {
			return "", err
		} else if len(result) > 0 {
			return result, nil
		}

		// Read more input from the tty. This is slightly complicated in that we need to
		// preserve the data in p.inBytes which may be a partial escape sequence.
		if len(p.inBytes) > 0 {
			n := copy(p.inBuf[:], p.inBytes)
			p.inBytes = p.inBuf[:n]
		}
		readBuf := p.inBuf[len(p.inBytes):]

		p.mu.Unlock()
		n, err := p.in.Read(readBuf)
		p.mu.Lock()

		if err != nil {
			return "", err
		}
		p.inBytes = p.inBuf[:n+len(p.inBytes)]
	}
}

func (p *Prompt) processInputLocked() (string, error) {
	var err error
	for err == nil {
		var key rune
		origInBytes := p.inBytes
		key, p.inBytes = parseKey(p.inBytes)
		if key == utf8.RuneError {
			break
		}
		debugPrintf(" input: %q -> %s\n",
			origInBytes[:len(origInBytes)-len(p.inBytes)], debugKey(key))
		err = p.dispatchKeyLocked(key)
	}

	if err == nil || errors.Is(err, io.EOF) {
		// Flush any buffered rendering commands.
		p.mu.state.screen.Flush(p.out)
	}

	if errors.Is(err, io.EOF) {
		if text := string(p.mu.state.screen.Text()); len(text) > 0 {
			p.mu.state.history.Add(text)
			return text, nil
		}
	}
	return "", err
}

func (p *Prompt) updateSize() error {
	if p.fd == -1 {
		return nil
	}

	width, height, err := term.GetSize(p.fd)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.mu.state.screen.SetSize(width, height)
	p.mu.state.screen.Flush(p.out)
	return nil
}

func (p *Prompt) dispatchKeyLocked(key rune) error {
	s := &p.mu.state
	cmd := p.bindings[key]
	if cmd == "" {
		cmd = cmdInsertChar
	}

	if ok, err := s.killRing.Dispatch(s, cmd, key); err != nil {
		return err
	} else if ok {
		return nil
	}

	if ok, err := s.history.Dispatch(s, cmd, key); err != nil {
		return err
	} else if ok {
		return nil
	}

	if fn, ok := baseCommands[cmd]; ok {
		_, err := fn(s, key)
		return err
	}

	return nil
}
