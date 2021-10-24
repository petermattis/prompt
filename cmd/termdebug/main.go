package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func debugCopy(dst io.Writer, src io.Reader, debug io.Writer, name string) {
	buf := make([]byte, 4096)
	for {
		nr, errR := src.Read(buf)
		if nr > 0 {
			fmt.Fprintf(debug, "%s: %q\n", name, buf[:nr])
			nw, errW := dst.Write(buf[:nr])
			if nw < 0 || nr < nw {
				fmt.Fprintf(debug, "%s: invalid write (nr=%d, nw=%d)\n", name, nr, nw)
			}
			if errW != nil {
				fmt.Fprintf(debug, "%s: write error: %+v\n", name, errW)
				break
			}
			if nr != nw {
				fmt.Fprintf(debug, "%s: short write (nr=%d, nw=%d)\n", name, nr, nw)
				break
			}
		}
		if errR != nil {
			if errR != io.EOF {
				fmt.Fprintf(debug, "%s: read error: %+v\n", name, errR)
			}
			break
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <command> [<args>]\n", os.Args[0])
		os.Exit(1)
	}

	c := exec.Command(os.Args[1], os.Args[2:]...)

	debug, err := os.Create("debug.txt")
	if err != nil {
		panic(err)
	}
	defer debug.Close()

	// Start the command with a pty.
	ptmx, err := pty.Start(c)
	if err != nil {
		panic(err)
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.
	go func() {
		debugCopy(ptmx, os.Stdin, debug, "stdin")
	}()

	debugCopy(os.Stdout, ptmx, debug, "stdout")
}
