package prompt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestParseKey(t *testing.T) {
	var sequences = map[string]rune{
		"\x7f":      keyBackspace,
		"a":         rune('a'),
		"b":         rune('b'),
		"«":         rune('«'),
		"»":         rune('»'),
		"\x1bb":     rune('b') | keyAlt,
		"\x1bf":     rune('f') | keyAlt,
		"\x1b«":     rune('«') | keyAlt,
		"\x1b»":     rune('»') | keyAlt,
		"\x01":      keyCtrlA,
		"\x02":      keyCtrlB,
		"\x05":      keyCtrlE,
		"\x06":      keyCtrlF,
		"\x08":      keyCtrlH,
		"\x0b":      keyCtrlK,
		"\x0c":      keyCtrlL,
		"\x10":      keyCtrlP,
		"\x17":      keyCtrlW,
		"\x1bOA":    keyUp,
		"\x1bOB":    keyDown,
		"\x1bOC":    keyRight,
		"\x1bOD":    keyLeft,
		"\x1bOH":    keyHome,
		"\x1bOF":    keyEnd,
		"\x1bOa":    keyUp | keyCtrl,
		"\x1bOb":    keyDown | keyCtrl,
		"\x1bOc":    keyRight | keyCtrl,
		"\x1bOd":    keyLeft | keyCtrl,
		"\x1b[A":    keyUp,
		"\x1b[B":    keyDown,
		"\x1b[C":    keyRight,
		"\x1b[D":    keyLeft,
		"\x1b[H":    keyHome,
		"\x1b[F":    keyEnd,
		"\x1b[1;3A": keyUp | keyAlt,
		"\x1b[1;3B": keyDown | keyAlt,
		"\x1b[1;3C": keyRight | keyAlt,
		"\x1b[1;3D": keyLeft | keyAlt,
		"\x1b[1;9A": keyUp | keyAlt,
		"\x1b[1;9B": keyDown | keyAlt,
		"\x1b[1;9C": keyRight | keyAlt,
		"\x1b[1;9D": keyLeft | keyAlt,
		"\x1b[1;5A": keyUp | keyCtrl,
		"\x1b[1;5B": keyDown | keyCtrl,
		"\x1b[1;5C": keyRight | keyCtrl,
		"\x1b[1;5D": keyLeft | keyCtrl,
		"\x1b[1~":   keyHome,
		"\x1b[200~": keyPasteStart,
		"\x1b[201~": keyPasteEnd,
		"\x1b[3~":   keyDelete,
		"\x1b[4~":   keyEnd,
		"\x1b[5~":   keyPageUp,
		"\x1b[6~":   keyPageDown,
		"\x1b[7~":   keyHome,
		"\x1b[8~":   keyEnd,
	}

	incomplete := map[string]rune{
		"":          utf8.RuneError,
		"\x1b":      utf8.RuneError,
		"\x1b[G":    keyUnknown,
		"\x1b[10":   utf8.RuneError,
		"\x1b[1;":   utf8.RuneError,
		"\x1b[1;3E": keyUnknown,
		"\x1b[1;5E": keyUnknown,
		"\x1b[9":    utf8.RuneError,
	}

	for seq, key := range sequences {
		k, _ := parseKey([]byte(seq))
		require.Equalf(t, key, k, "%q", seq)

		// An escape prefix on an escape sequence will add the keyAlt modifier.
		seq = "\x1b" + seq
		k, _ = parseKey([]byte(seq))
		if key != keyPasteStart && key != keyPasteEnd {
			key |= keyAlt
		}
		require.Equalf(t, key, k, "%q", seq)
	}

	for seq, key := range incomplete {
		k, _ := parseKey([]byte(seq))
		require.Equal(t, key, k, "%q", seq)
	}
}

func TestInputSupportedTerms(t *testing.T) {
	t.Skip("not really a test, unskip to recompute the number of supported terminals")

	const termInfoDir = "/usr/share/terminfo"

	// capRE extracts the capabilities from the infocmp output. Note that we only
	// support control sequences that begin with "\E[" (the standard Control
	// Sequence Introducer) or "\EO" (the introducer used by DEC terminals for some
	// keys and then somehow copied to lots of unrelated terminals).
	capRE := regexp.MustCompile(`(\bkey_\w+)=(\\E[\[O][^,]*),`)

	// A map from terminfo capability name to the Go const name we'll output when
	// the key's control sequence is matched.
	capToKey := map[string]rune{
		"key_dc":    keyDelete,
		"key_down":  keyDown,
		"key_end":   keyEnd,
		"key_home":  keyHome,
		"key_left":  keyLeft,
		"key_npage": keyPageDown,
		"key_ppage": keyPageUp,
		"key_right": keyRight,
		"key_up":    keyUp,
	}

	supportedTerms := make(map[string]struct{})
	unsupportedTerms := make(map[string]struct{})

	processLine := func(term string, line []byte) {
		m := capRE.FindSubmatch(line)
		if len(m) == 0 {
			return
		}
		key, seq := string(m[1]), string(m[2])
		seq = strings.ReplaceAll(seq, "\\E", "\x1b")
		if supportedSeqs[seq] == capToKey[key] {
			return
		}
		unsupportedTerms[term] = struct{}{}
	}

	processTermInfo := func(term string) {
		c := exec.Command("infocmp", "-L1", "-A", termInfoDir, term)
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("infocmp failed: %+v %s\n%s", err, c.Args, out)
		}

		for buf := bytes.NewBuffer(out); ; {
			line, err := buf.ReadBytes('\n')
			if err != nil && !errors.Is(err, io.EOF) {
				t.Fatalf("%s: %+v\n", term, err)
			}

			processLine(term, line)

			if errors.Is(err, io.EOF) {
				break
			}
		}

		if _, ok := unsupportedTerms[term]; !ok {
			supportedTerms[term] = struct{}{}
		}
	}

	err := filepath.WalkDir(termInfoDir, func(path string, d fs.DirEntry, err error) error {
		if d.Type().IsRegular() {
			processTermInfo(filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unable to find terminfos: %+v", err)
	}

	fmt.Fprintf(os.Stderr, "%4d supported terms\n", len(supportedTerms))
	fmt.Fprintf(os.Stderr, "%4d unsupported terms\n", len(unsupportedTerms))
}
