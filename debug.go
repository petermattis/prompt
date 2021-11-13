package prompt

import (
	"fmt"
	"io"
	"os"
	"sync"
	"unicode/utf8"
)

var dbg = struct {
	sync.Once
	w   io.WriteCloser
	err error
}{}

func initDebug() {
	path := os.Getenv("PROMPT_DEBUG")
	if path == "" {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		dbg.err = err
		return
	}
	dbg.w = f
}

func debugPrintf(format string, args ...interface{}) {
	dbg.Do(initDebug)
	if dbg.w == nil {
		return
	}
	fmt.Fprintf(dbg.w, format, args...)
}

func debugKey(r rune) string {
	if r < 32 {
		return "Control-" + string(rune(r+0x60))
	}

	var s string
	switch b := r & ^(keyAlt | keyCtrl); b {
	case utf8.RuneError:
		s = "<incomplete>"
	case keyBackspace:
		s = "<backspace>"
	case keyUnknown:
		s = "<unknown>"
	case keyUp:
		s = "<up>"
	case keyDown:
		s = "<down>"
	case keyLeft:
		s = "<left>"
	case keyRight:
		s = "<right>"
	case keyHome:
		s = "<home>"
	case keyEnd:
		s = "<end>"
	case keyPageUp:
		s = "<page-up>"
	case keyPageDown:
		s = "<page-down>"
	case keyDelete:
		s = "<delete>"
	case keyPasteStart:
		s = "<paste-start>"
	case keyPasteEnd:
		s = "<paste-end>"
	default:
		s = string(b)
	}

	if (r & keyAlt) != 0 {
		s = "Meta-" + s
	}
	if (r & keyCtrl) != 0 {
		s = "Control-" + s
	}
	return s
}
