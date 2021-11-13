package prompt

import "unicode/utf8"

const (
	keyCtrlA     = 1
	keyCtrlB     = 2
	keyCtrlC     = 3
	keyCtrlD     = 4
	keyCtrlE     = 5
	keyCtrlF     = 6
	keyCtrlG     = 7
	keyCtrlH     = 8
	keyCtrlK     = 11
	keyCtrlL     = 12
	keyCtrlN     = 14
	keyCtrlP     = 16
	keyCtrlR     = 18
	keyCtrlS     = 19
	keyCtrlT     = 20
	keyCtrlU     = 21
	keyCtrlW     = 23
	keyCtrlY     = 25
	keyEnter     = '\r'
	keyEscape    = 27
	keyBackspace = 127
	keyUnknown   = 0xd800 /* UTF-16 surrogate area */ + iota
	keyUp
	keyDown
	keyLeft
	keyRight
	keyHome
	keyEnd
	keyPageUp
	keyPageDown
	keyDelete
	keyPasteStart
	keyPasteEnd
	keyCtrl = 0x20000000
	keyAlt  = 0x40000000
)

// A map of the supported control sequences to the Go code that will be emitted
// when the control sequence is matched.
//
// Note that we can't specify control sequences to cover the desired key input
// for all terminals because the same control sequence is sometimes used by
// different terminals to represent different keys. The control sequences below
// support 75% of the ~2500 terminals listed in my terminfo database.
var supportedSeqs = map[string]rune{
	"\x1b[3~":   keyDelete,
	"\x1bOB":    keyDown,
	"\x1b[B":    keyDown,
	"\x1bOb":    keyDown | keyCtrl,
	"\x1b[1;5B": keyDown | keyCtrl,
	"\x1b[1;3B": keyDown | keyAlt,
	"\x1b[1;9B": keyDown | keyAlt,
	"\x1bOF":    keyEnd,
	"\x1b[F":    keyEnd,
	"\x1b[4~":   keyEnd,
	"\x1b[8~":   keyEnd,
	"\x1bOH":    keyHome,
	"\x1b[H":    keyHome,
	"\x1b[1~":   keyHome,
	"\x1b[7~":   keyHome,
	"\x1bOD":    keyLeft,
	"\x1b[D":    keyLeft,
	"\x1bOd":    keyLeft | keyCtrl,
	"\x1b[1;5D": keyLeft | keyCtrl,
	"\x1b[1;3D": keyLeft | keyAlt,
	"\x1b[1;9D": keyLeft | keyAlt,
	"\x1b[6~":   keyPageDown,
	"\x1b[5~":   keyPageUp,
	"\x1b[200~": keyPasteStart,
	"\x1b[201~": keyPasteEnd,
	"\x1bOC":    keyRight,
	"\x1b[C":    keyRight,
	"\x1bOc":    keyRight | keyCtrl,
	"\x1b[1;5C": keyRight | keyCtrl,
	"\x1b[1;3C": keyRight | keyAlt,
	"\x1b[1;9C": keyRight | keyAlt,
	"\x1bOA":    keyUp,
	"\x1b[A":    keyUp,
	"\x1bOa":    keyUp | keyCtrl,
	"\x1b[1;5A": keyUp | keyCtrl,
	"\x1b[1;3A": keyUp | keyAlt,
	"\x1b[1;9A": keyUp | keyAlt,
}

type seqTrie struct {
	children []seqTrie
	key      byte
	value    rune
}

func (t *seqTrie) findChild(b byte) *seqTrie {
	for i := range t.children {
		child := &t.children[i]
		if child.key == b {
			return child
		}
	}
	return nil
}

func (t *seqTrie) add(seq []byte, value rune) {
	node := t
	for _, b := range seq {
		child := node.findChild(b)
		if child == nil {
			node.children = append(node.children, seqTrie{key: b})
			child = &node.children[len(node.children)-1]
		}
		node = child
	}
	node.value = value
}

func (t *seqTrie) match(buf, origBuf []byte, mods rune) (rune, []byte) {
	node := t
	for i, b := range buf {
		node = node.findChild(b)
		if node == nil {
			// If we get here then we have a sequence that we don't recognise, or a partial
			// sequence. It's not clear how one should find the end of a sequence without
			// knowing them all, but it seems that [a-zA-Z~] only appears at the end of a
			// sequence.
			for j := i; j < len(buf); j++ {
				b := buf[j]
				if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b == '~' {
					return keyUnknown, buf[i+1:]
				}
			}
			return utf8.RuneError, origBuf
		}
		if len(node.children) == 0 {
			// We've reached a leaf node, so return the value.
			//
			// Special case handling for the keyPasteStart and keyPasteEnd sequences: we
			// don't include the supplied modifiers.
			if node.value == keyPasteStart || node.value == keyPasteEnd {
				mods = 0
			}
			return node.value | mods, buf[i+1:]
		}
	}
	// We were matching a sequence but ran out of bytes in the supplied buffer.
	// Return an error so the caller knows to read more and try again.
	return utf8.RuneError, origBuf
}

var seqMatcher = func() *seqTrie {
	t := &seqTrie{}
	for seq, value := range supportedSeqs {
		t.add([]byte(seq), value)
	}
	return t
}()

// parseKey parses a single key from the prefix of the specified byte slice.
// Parsing keys is challenging because the input sequences used by terminals
// differ. Rather than the termcap/terminfo approach of determining the input
// sequences based on the $TERM env var, this code takes the approach of
// handling the most common sequences used by the majority (~75%) of terminals
// and all modern terminals. This is also the approached used by linenoise, and
// libraries inspired by linenoise.
//
// If the input sequence is not recognized, keyUnknown is returned. If a prefix
// of a recognized input sequence is matched but there are insufficient bytes in
// the input, utf8.RuneError is returned. On success, the remaining bytes in the
// input will be returned.
//
// See https://invisible-island.net/xterm/xterm-function-keys.html which
// describes the xterm function keys, and also points to dumping term output
// using infocmp.
//
// See https://en.wikipedia.org/wiki/ANSI_escape_code#Terminal_input_sequences
// which describes the general structure of terminal input sequences.
func parseKey(buf []byte) (rune, []byte) {
	var origBuf = buf
	var mods rune

	for len(buf) >= 2 {
		// An escape that is not the beginning of "\x1bO..." or "\x1b[..." sets the keyAlt
		// modifier.
		if buf[0] != keyEscape || buf[1] == 'O' || buf[1] == '[' {
			break
		}
		mods |= keyAlt
		buf = buf[1:]
	}

	if len(buf) <= 0 {
		return utf8.RuneError, origBuf
	}

	if buf[0] != keyEscape {
		if !utf8.FullRune(buf) {
			return utf8.RuneError, origBuf
		}
		r, l := utf8.DecodeRune(buf)
		return r | mods, buf[l:]
	}

	return seqMatcher.match(buf, origBuf, mods)
}
