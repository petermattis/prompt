package prompt

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TODO(peter): Support the "\C-<char>" and "\M-<char>" syntax?
// TODO(peter): Support multi-key commands (e.g. (C-x C-x))?

type command string

const (
	cmdAbort                 command = "abort"
	cmdBackwardChar                  = "backward-char"
	cmdBackwardDeleteChar            = "backward-delete-char"
	cmdBackwardKillLine              = "backward-kill-line"
	cmdBackwardKillWord              = "backward-kill-word"
	cmdBackwardWord                  = "backward-word"
	cmdBeginningOfLine               = "beginning-of-line"
	cmdCancel                        = "cancel"
	cmdClearScreen                   = "clear-screen"
	cmdDeleteChar                    = "delete-char"
	cmdDeleteHorizontalSpace         = "delete-horizontal-space"
	cmdEndOfLine                     = "end-of-line"
	cmdEnter                         = "enter"
	cmdExitOrDeleteChar              = "exit-or-delete-char"
	cmdFinishOrEnter                 = "finish-or-enter"
	cmdForwardChar                   = "forward-char"
	cmdForwardSearchHistory          = "forward-search-history"
	cmdForwardWord                   = "forward-word"
	cmdInsertChar                    = "insert-char"
	cmdKillLine                      = "kill-line"
	cmdKillWord                      = "kill-word"
	cmdNextHistory                   = "next-history"
	cmdPreviousHistory               = "previous-history"
	cmdReverseSearchHistory          = "reverse-search-history"
	cmdSetMark                       = "set-mark"
	cmdTransposeChars                = "transpose-chars"
	cmdTransposeWords                = "transpose-words"
	cmdUndo                          = "undo"
	cmdYank                          = "yank"
	cmdYankPop                       = "yank-pop"
)

const defaultBindings = string(`
bind Backspace       ` + cmdBackwardDeleteChar + `
bind Delete          ` + cmdDeleteChar + `
bind Down            ` + cmdNextHistory + `
bind End             ` + cmdEndOfLine + `
bind Enter           ` + cmdFinishOrEnter + `
bind Home            ` + cmdBeginningOfLine + `
bind Left            ` + cmdBackwardChar + `
bind Right           ` + cmdForwardChar + `
bind Up              ` + cmdPreviousHistory + `
bind Control-Left    ` + cmdBackwardWord + `
bind Control-Right   ` + cmdForwardWord + `
bind Control-Space   ` + cmdSetMark + `
bind Control-_       ` + cmdUndo + `
bind Control-a       ` + cmdBeginningOfLine + `
bind Control-b       ` + cmdBackwardChar + `
bind Control-c       ` + cmdCancel + `
bind Control-d       ` + cmdExitOrDeleteChar + `
bind Control-e       ` + cmdEndOfLine + `
bind Control-f       ` + cmdForwardChar + `
bind Control-g       ` + cmdAbort + `
bind Control-h       ` + cmdBackwardDeleteChar + `
bind Control-k       ` + cmdKillLine + `
bind Control-l       ` + cmdClearScreen + `
bind Control-n       ` + cmdNextHistory + `
bind Control-p       ` + cmdPreviousHistory + `
bind Control-r       ` + cmdReverseSearchHistory + `
bind Control-s       ` + cmdForwardSearchHistory + `
bind Control-t       ` + cmdTransposeChars + `
bind Control-u       ` + cmdBackwardKillLine + `
bind Control-w       ` + cmdBackwardKillWord + `
bind Control-y       ` + cmdYank + `
bind Meta-Backspace  ` + cmdBackwardKillWord + `
bind Meta-Control-h  ` + cmdBackwardKillWord + `
bind Meta-Enter      ` + cmdEnter + `
bind Meta-Left       ` + cmdBackwardWord + `
bind Meta-Right      ` + cmdForwardWord + `
bind Meta-\          ` + cmdDeleteHorizontalSpace + `
bind Meta-b          ` + cmdBackwardWord + `
bind Meta-d          ` + cmdKillWord + `
bind Meta-f          ` + cmdForwardWord + `
bind Meta-t          ` + cmdTransposeWords + `
bind Meta-y          ` + cmdYankPop + `
`)

var commandAliases = map[string]command{
	"unix-line-discard": cmdBackwardKillLine,
}

var namedKeys = map[string]rune{
	"backspace": keyBackspace,
	"delete":    keyDelete,
	"down":      keyDown,
	"end":       keyEnd,
	"enter":     keyEnter,
	"home":      keyHome,
	"left":      keyLeft,
	"page-down": keyPageDown,
	"page-up":   keyPageUp,
	"right":     keyRight,
	"space":     ' ',
	"up":        keyUp,
}

type commandFunc func(s *state, key rune) (bool, error)

var baseCommands = map[command]commandFunc{
	cmdBackwardChar: func(s *state, key rune) (bool, error) {
		// Move to the beginning of the previous grapheme.
		s.screen.MoveTo(s.screen.PrevGraphemeStart())
		return true, nil
	},
	cmdBackwardDeleteChar: func(s *state, key rune) (bool, error) {
		// Erase to the beginning of the previous grapheme.
		s.screen.EraseTo(s.screen.PrevGraphemeStart())
		return true, nil
	},
	cmdBackwardWord: func(s *state, key rune) (bool, error) {
		// Move to the beginning of the previous word.
		s.screen.MoveTo(s.screen.PrevWordStart(s.screen.Position()))
		return true, nil
	},
	cmdBeginningOfLine: func(s *state, key rune) (bool, error) {
		// Move to beginning of input text.
		s.screen.MoveTo(0)
		return true, nil
	},
	cmdCancel: func(s *state, key rune) (bool, error) {
		if len(s.screen.Text()) == 0 {
			return true, io.EOF
		}
		// Cancel the current input, but leave it on screen.
		s.screen.Cancel()
		return true, nil
	},
	cmdClearScreen: func(s *state, key rune) (bool, error) {
		// Erases the screen, moves the cursor to the home position, and redraws the
		// prompt and input text.
		s.screen.Refresh()
		return true, nil
	},
	cmdDeleteChar: func(s *state, key rune) (bool, error) {
		// Delete the next grapheme.
		s.screen.EraseTo(s.screen.NextGraphemeEnd())
		return true, nil
	},
	cmdDeleteHorizontalSpace: func(s *state, key rune) (bool, error) {
		// Delete all whitespace around the current position.
		text := s.screen.Text()
		prevWordEnd := s.screen.Position()
		for ; prevWordEnd > 0; prevWordEnd-- {
			if !unicode.IsSpace(text[prevWordEnd-1]) {
				break
			}
		}
		nextWordStart := prevWordEnd
		for ; nextWordStart < len(text); nextWordStart++ {
			if !unicode.IsSpace(text[nextWordStart]) {
				break
			}
		}
		if nextWordStart >= s.screen.Position() && prevWordEnd < nextWordStart {
			s.screen.MoveTo(prevWordEnd)
			s.screen.EraseTo(nextWordStart)
		}
		return true, nil
	},
	cmdEndOfLine: func(s *state, key rune) (bool, error) {
		// Move to end of input text.
		s.screen.MoveTo(s.screen.End())
		return true, nil
	},
	cmdEnter: func(s *state, key rune) (bool, error) {
		s.screen.Insert('\n')
		return true, nil
	},
	cmdExitOrDeleteChar: func(s *state, key rune) (bool, error) {
		if len(s.screen.Text()) == 0 {
			return true, io.EOF
		}
		// Delete the next grapheme.
		s.screen.EraseTo(s.screen.NextGraphemeEnd())
		return true, nil
	},
	cmdFinishOrEnter: func(s *state, key rune) (bool, error) {
		if s.inputFinished == nil || s.inputFinished(string(s.screen.Text())) {
			s.screen.outbuf.WriteString("\r\n")
			return true, io.EOF
		}
		s.screen.Insert('\n')
		return true, nil
	},
	cmdForwardChar: func(s *state, key rune) (bool, error) {
		// Move to the end of the next grapheme.
		s.screen.MoveTo(s.screen.NextGraphemeEnd())
		return true, nil
	},
	cmdForwardWord: func(s *state, key rune) (bool, error) {
		// Move to the end of the next word.
		s.screen.MoveTo(s.screen.NextWordEnd(s.screen.Position()))
		return true, nil
	},
	cmdInsertChar: func(s *state, key rune) (bool, error) {
		// Insert the character at the current cursor position.
		s.screen.Insert(key)
		return true, nil
	},
	cmdSetMark: func(s *state, key rune) (bool, error) {
		// TODO(peter): set-mark
		// - The mark is a logical position in the text. If text is inserted or erased
		//   before the mark, the mark's position is adjusted.
		return true, nil
	},
	cmdTransposeChars: func(s *state, key rune) (bool, error) {
		// Transpose the previous grapheme with the next grapheme.
		if text := s.screen.EraseTo(s.screen.PrevGraphemeStart()); len(text) > 0 {
			s.screen.MoveTo(s.screen.NextGraphemeEnd())
			s.screen.Insert([]rune(text)...)
		}
		return true, nil
	},
	cmdTransposeWords: func(s *state, key rune) (bool, error) {
		// Transpose the previous word with the next word.
		nextWordEnd := s.screen.NextWordEnd(s.screen.Position())
		nextWordStart := s.screen.PrevWordStart(nextWordEnd)
		prevWordStart := s.screen.PrevWordStart(nextWordStart)
		prevWordEnd := s.screen.NextWordEnd(prevWordStart)
		if prevWordStart != nextWordStart {
			s.screen.MoveTo(nextWordStart)
			nextWord := s.screen.EraseTo(nextWordEnd)
			s.screen.MoveTo(prevWordStart)
			prevWord := s.screen.EraseTo(prevWordEnd)
			s.screen.Insert([]rune(nextWord)...)
			s.screen.MoveTo(s.screen.Position() + nextWordStart - prevWordEnd)
			s.screen.Insert([]rune(prevWord)...)
		}
		return true, nil
	},
	cmdUndo: func(s *state, key rune) (bool, error) {
		// TODO(peter): Undo. Each input buffer needs to maintain a series of edits
		// (insertions, and deletions). The history entries then become these input
		// buffers rather than just strings. Undo only applies to the current line. A
		// series of insertions creates a single edit.
		return true, nil
	},
}

func isValidCommand(cmd command) bool {
	if _, ok := baseCommands[cmd]; ok {
		return true
	}
	if _, ok := killCommands[cmd]; ok {
		return true
	}
	if _, ok := yankCommands[cmd]; ok {
		return true
	}
	if _, ok := historyCommands[cmd]; ok {
		return true
	}
	return false
}

func parseBinding(binding string) (key rune, cmd command, err error) {
	const (
		controlPrefix = "Control-"
		metaPrefix    = "Meta-"
	)

	parts := strings.Fields(binding)
	if len(parts) != 3 || parts[0] != "bind" {
		return utf8.RuneError, "", fmt.Errorf("invalid binding: [%s]", binding)
	}

	cmd = command(parts[2])
	if s, ok := commandAliases[string(cmd)]; ok {
		cmd = s
	}
	if !isValidCommand(cmd) {
		return utf8.RuneError, "", fmt.Errorf("unknown command: %s", cmd)
	}

	origKey := parts[1]
	var mods rune
	for s := parts[1]; len(s) > 0; {
		if strings.HasPrefix(s, controlPrefix) {
			if (mods & keyCtrl) != 0 {
				return utf8.RuneError, "", fmt.Errorf("invalid key: %q", origKey)
			}
			mods |= keyCtrl
			s = s[len(controlPrefix):]
			continue
		}
		if strings.HasPrefix(s, metaPrefix) {
			if (mods & keyAlt) != 0 {
				return utf8.RuneError, "", fmt.Errorf("invalid key: %q", origKey)
			}
			mods |= keyAlt
			s = s[len(metaPrefix):]
			continue
		}
		if key = namedKeys[strings.ToLower(s)]; key == 0 {
			var l int
			key, l = utf8.DecodeRuneInString(s)
			if l != len(s) {
				return utf8.RuneError, "", fmt.Errorf("invalid key: %q", origKey)
			}
		}
		break
	}

	// Translate C-[a-z] into keyCtrl[A-Z].
	if (mods & keyCtrl) != 0 {
		if key >= 'a' && key <= ('a'+31) {
			key -= 0x60
			mods ^= keyCtrl
		}
	}

	return key | mods, cmd, nil
}

func parseBindings(m map[rune]command, data string) error {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		key, cmd, err := parseBinding(line)
		if err != nil {
			return err
		}
		m[key] = cmd
		if (key & keyAlt) != 0 {
			b := key & ^(keyAlt | keyCtrl)
			switch {
			case unicode.IsLower(b):
				b = unicode.ToUpper(b)
			case unicode.IsUpper(b):
				b = unicode.ToLower(b)
			}
			key = b | (key & (keyAlt | keyCtrl))
			m[key] = cmd
		}
	}
	return nil
}
