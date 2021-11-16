package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

var historyCommands = map[command]commandFunc{
	cmdAbort: func(s *state, key rune) (bool, error) {
		return s.history.AbortSearch(s)
	},
	cmdBackwardDeleteChar: func(s *state, key rune) (bool, error) {
		return s.history.TruncateSearchKey(s)
	},
	cmdCancel: func(s *state, key rune) (bool, error) {
		return s.history.CancelSearch(s)
	},
	cmdForwardSearchHistory: func(s *state, key rune) (bool, error) {
		return s.history.ForwardSearch(s)
	},
	cmdInsertChar: func(s *state, key rune) (bool, error) {
		return s.history.AppendSearchKey(s, key)
	},
	cmdReverseSearchHistory: func(s *state, key rune) (bool, error) {
		return s.history.ReverseSearch(s)
	},
	cmdNextHistory: func(s *state, key rune) (bool, error) {
		return s.history.Next(s)
	},
	cmdPreviousHistory: func(s *state, key rune) (bool, error) {
		return s.history.Previous(s)
	},
}

// history implements a fixed size circular list of history entries and commands
// for navigating and searching the list. Adjacent duplicate history entries are
// suppressed. Forward and reverse incremental search of both history entries
// and the pending input including positioning of the cursor within the
// currently matched line when there is more than one match on a line.
type history struct {
	path             string
	file             io.WriteCloser
	pending          string
	entries          []string
	head             int
	size             int
	maxSize          int
	index            int
	searchDir        int
	searchMatched    bool
	searchKey        string
	searchMatchedKey string
}

// Load loads history entries from file. The history entries are expected to be
// encoded one per line in the file with whitespace encoded as \000 octal
// escapes. The first entry must be "_HiStOrY_V2_".
func (h *history) Load() error {
	// This is the special value stored at the start of libedit history files.
	const historyCookie = "_HiStOrY_V2_"

	if h.path == "" {
		return nil
	}

	f, err := os.OpenFile(h.path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	var n int
	for s := bufio.NewScanner(f); s.Scan(); n++ {
		text := s.Text()
		if n == 0 {
			if text != historyCookie {
				return fmt.Errorf("malformed history cookie: %q != %q", text, historyCookie)
			}
			continue
		}
		v, err := decodeVis(text)
		if err != nil {
			return err
		}
		h.Add(v)
	}

	if count := n - 1; count < 0 {
		// If the history file was empty, write a cookie to initialize it.
		fmt.Fprintf(f, "%s\n", historyCookie)
	} else if count > (h.maxSize*5)/4 {
		// The history file is 25% large than the max size, rewrite it.
		f.Close()
		f, err = os.OpenFile(h.path, os.O_CREATE|os.O_RDWR|os.O_APPEND|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		fmt.Fprintf(f, "%s\n", historyCookie)
		for i := len(h.entries) - 1; i >= 0; i-- {
			fmt.Fprintf(f, "%s\n", encodeVis(h.entry(i)))
		}
	}

	h.file, f = f, nil
	return nil
}

// Close closes the history file (if one is open).
func (h *history) Close() error {
	if h.file != nil {
		if err := h.file.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Add adds a new entry to history, overwriting the oldest entry if the max
// number of history entries has been reached. The current index in the history
// navigation is reset.
func (h *history) Add(s string) {
	if h.maxSize == 0 {
		// History is disabled.
		debugPrintf("history: disabled\n")
		return
	}
	if h.entry(0) == s {
		// Don't add a new entry if it is identical to the previous entry.
		debugPrintf("history: elide duplicate\n")
		return
	}
	if h.maxSize == -1 || len(h.entries) < h.maxSize {
		h.entries = append(h.entries, "")
	}
	h.head = (h.head + 1) % len(h.entries)
	h.entries[h.head] = s
	h.index = -1

	// If we have a history file, append the new entry.
	if h.file != nil {
		fmt.Fprintf(h.file, "%s\n", encodeVis(s))
	}
}

// Next saves the current history entry, advances to the next entry, and sets
// that entry as the input text. If history search is active next advances to
// the next forward search result.
func (h *history) Next(s *state) (bool, error) {
	if h.searchDir != 0 {
		return h.ForwardSearch(s)
	}
	if h.index == -1 {
		return false, nil
	}
	h.save(s.screen.Text())
	h.index--
	s.screen.MoveTo(0)
	s.screen.EraseTo(s.screen.End())
	s.screen.Insert([]rune(h.entry(h.index))...)
	return true, nil
}

// Previous saves the current history entry, advances to the previous history
// entry, and sets that entry as the input text. If history search is active
// next advances to the next search reverse search result.
func (h *history) Previous(s *state) (bool, error) {
	if h.searchDir != 0 {
		return h.ReverseSearch(s)
	}
	if h.index+1 >= len(h.entries) {
		return false, nil
	}
	h.save(s.screen.Text())
	h.index++
	s.screen.MoveTo(0)
	s.screen.EraseTo(s.screen.End())
	s.screen.Insert([]rune(h.entry(h.index))...)
	return true, nil
}

// AbortSearch resets the search key to the last search key which matched if the
// last search failed to match. Otherwise, cancels history search if active,
// restoring normal line editing.
func (h *history) AbortSearch(s *state) (bool, error) {
	if h.searchDir == 0 {
		return false, nil
	}
	if !h.searchMatched {
		h.searchKey = h.searchMatchedKey
		h.updateSearch(s, false /* advance */)
		return true, nil
	}
	return h.CancelSearch(s)
}

// CancelSearch cancels history search if active, restoring normal line editing.
func (h *history) CancelSearch(s *state) (bool, error) {
	if h.searchDir == 0 {
		return false, nil
	}
	s.screen.SetSuffix(nil)
	h.searchDir = 0
	h.searchMatched = false
	h.searchKey = ""
	h.searchMatchedKey = ""
	return true, nil
}

// ForwardSearch starts history search if inactive, and switches to forward
// search.
func (h *history) ForwardSearch(s *state) (bool, error) {
	h.maybeInitSearch(s)
	h.searchDir = +1
	h.updateSearch(s, true /* advance */)
	return true, nil
}

// ReverseSearch starts history search if inactive, and switches to reverse
// search.
func (h *history) ReverseSearch(s *state) (bool, error) {
	h.maybeInitSearch(s)
	h.searchDir = -1
	h.updateSearch(s, true /* advance */)
	return true, nil
}

// AppendSearchKey appends the specified character to the search key.
func (h *history) AppendSearchKey(s *state, key rune) (bool, error) {
	if h.searchDir == 0 {
		return false, nil
	}
	if isPrintable(key) {
		h.searchKey += string(key)
		h.updateSearch(s, false /* advance */)
	}
	return true, nil
}

// TruncateSearchKey trims the last character from the search key.
func (h *history) TruncateSearchKey(s *state) (bool, error) {
	if h.searchDir == 0 {
		return false, nil
	}
	if len(h.searchKey) > 0 {
		_, size := utf8.DecodeLastRuneInString(h.searchKey)
		h.searchKey = h.searchKey[:len(h.searchKey)-size]
		h.updateSearch(s, false /* advance */)
	}
	return true, nil
}

// Dispatch processes the specified command. Non-history commands cause any
// history search to be aborted.
func (h *history) Dispatch(s *state, cmd command, key rune) (ok bool, err error) {
	if fn, ok := historyCommands[cmd]; ok {
		return fn(s, key)
	}
	if _, err := h.CancelSearch(s); err != nil {
		return true, err
	}
	return false, nil
}

func (h *history) String() string {
	var buf strings.Builder
	buf.WriteString("[")
	for i := range h.entries {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(h.entry(i))
	}
	buf.WriteString("]")
	return buf.String()
}

func (h *history) entry(n int) string {
	if n == -1 {
		return h.pending
	}
	i := h.entryIndex(n)
	if i == -1 {
		return ""
	}
	return h.entries[i]
}

func (h *history) entryIndex(n int) int {
	if n >= len(h.entries) {
		return -1
	}
	index := h.head - n
	if index < 0 {
		index += len(h.entries)
	}
	return index
}

func (h *history) save(cur []rune) {
	if h.index == -1 {
		h.pending = string(cur)
		return
	}
	index := h.entryIndex(h.index)
	if index == -1 {
		return
	}
	h.entries[index] = string(cur)
}

func (h *history) searchEntry(s *state, i int, advance bool) bool {
	var pos int
	entry := h.entry(i)

	switch h.searchDir {
	case +1:
		var n int
		if i == h.index {
			n = s.screen.Position()
			if advance {
				n++
			}
			if n > len(entry) {
				n = len(entry)
			}
		}
		pos = strings.Index(entry[n:], h.searchKey)
		if pos != -1 {
			pos += n
		}

	case -1:
		n := len(entry)
		if i == h.index {
			n = s.screen.Position() + len(h.searchKey)
			if advance {
				n--
			}
			if n < 0 {
				n = 0
			}
			if n > len(entry) {
				n = len(entry)
			}
		}
		pos = strings.LastIndex(entry[:n], h.searchKey)
	}

	if pos == -1 {
		return false
	}

	h.save(s.screen.Text())
	h.index = i
	s.screen.MoveTo(0)
	s.screen.EraseTo(s.screen.End())
	s.screen.Insert([]rune(entry)...)
	s.screen.MoveTo(utf8.RuneCountInString(entry[:pos]))
	return true
}

func (h *history) updateSearch(s *state, advance bool) {
	h.searchMatched = false
	if len(h.searchKey) > 0 {
		switch h.searchDir {
		case +1:
			for i := h.index; i >= -1; i-- {
				if h.searchEntry(s, i, advance) {
					h.searchMatched = true
					h.searchMatchedKey = h.searchKey
					break
				}
			}

		case -1:
			for i := h.index; i < len(h.entries); i++ {
				if h.searchEntry(s, i, advance) {
					h.searchMatched = true
					h.searchMatchedKey = h.searchKey
					break
				}
			}
		}
	}

	dir := "fwd"
	if h.searchDir < 0 {
		dir = "bck"
	}

	matched := "?"
	if len(h.searchKey) == 0 || h.searchMatched {
		matched = ":"
	}

	newSuffix := fmt.Sprintf("\n%s%s`%s'", dir, matched, h.searchKey)
	s.screen.SetSuffix([]rune(newSuffix))
}

func (h *history) maybeInitSearch(s *state) {
	if h.searchDir != 0 {
		return
	}
	if len(h.entries) == 0 {
		h.index = -1
	}
	h.save(s.screen.Text())
	h.searchMatchedKey = ""
}
