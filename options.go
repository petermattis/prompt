package prompt

import (
	"io"
	"os"
)

// Option defines the interface for Prompt options.
type Option interface {
	apply(p *Prompt)
}

type ttyOption struct {
	tty *os.File
}

func (o *ttyOption) apply(p *Prompt) {
	p.fd = int(o.tty.Fd())
	p.in = o.tty
	p.out = o.tty
}

// WithTTY allows configuring a prompt with a different TTY than stdin/stdout.
func WithTTY(tty *os.File) Option {
	return &ttyOption{
		tty: tty,
	}
}

type inputOption struct {
	r io.Reader
}

func (o *inputOption) apply(p *Prompt) {
	p.in = o.r
}

// WithInput allows configuring the input reader for a Prompt. This option is
// primarily useful for tests.
func WithInput(r io.Reader) Option {
	return &inputOption{
		r: r,
	}
}

type outputOption struct {
	w io.Writer
}

func (o *outputOption) apply(p *Prompt) {
	p.out = o.w
}

// WithOutput allows configuring the output writer for a Prompt. This option is
// primarily useful for tests.
func WithOutput(w io.Writer) Option {
	return &outputOption{
		w: w,
	}
}

type sizeOption struct {
	width, height int
}

func (o *sizeOption) apply(p *Prompt) {
	p.mu.state.screen.SetSize(o.width, o.height)
}

// WithSize allows configuring the initial width and height of a Prompt.
// Typically, the width and height of the terminal are automatically determined.
// This option is primarily useful for tests in conjunction with the WithInput
// and WithOutput options.
func WithSize(width, height int) Option {
	return &sizeOption{
		width:  width,
		height: height,
	}
}

type inputFinishedOption struct {
	fn func(text string) bool
}

func (o inputFinishedOption) apply(p *Prompt) {
	p.mu.state.inputFinished = o.fn
}

// WithInputFinished allows configuring a callback that will be invoked when
// enter is pressed to determine if the input is considered complete or not. If
// the input is not complete, a newline is instead inserted into the input.
func WithInputFinished(fn func(text string) bool) Option {
	return inputFinishedOption{fn}
}
