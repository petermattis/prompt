package prompt

import "strings"

const killRingMax = 10

var killCommands = map[command]commandFunc{
	cmdBackwardKillLine: func(s *state, key rune) (bool, error) {
		// Erase to the beginning of the input.
		if e := s.screen.EraseTo(0); len(e) > 0 {
			s.killRing.Prepend(e)
		}
		return true, nil
	},
	cmdBackwardKillWord: func(s *state, key rune) (bool, error) {
		// Delete zero or more spaces and then one or more characters.
		if e := s.screen.EraseTo(s.screen.PrevWordStart(s.screen.Position())); len(e) > 0 {
			s.killRing.Prepend(e)
		}
		return true, nil
	},
	cmdKillLine: func(s *state, key rune) (bool, error) {
		// Delete everything from the current cursor position to the end of line.
		if e := s.screen.EraseTo(s.screen.End()); len(e) > 0 {
			s.killRing.Append(e)
		}
		return true, nil
	},
	cmdKillWord: func(s *state, key rune) (bool, error) {
		// TODO(peter): if a mark is set, kill-region.

		// Delete zero or more spaces and then one or more characters.
		if e := s.screen.EraseTo(s.screen.NextWordEnd(s.screen.Position())); len(e) > 0 {
			s.killRing.Append(e)
		}
		return true, nil
	},
}

var yankCommands = map[command]commandFunc{
	cmdYank: func(s *state, key rune) (bool, error) {
		s.screen.Insert(s.killRing.Yank()...)
		return true, nil
	},
	cmdYankPop: func(s *state, key rune) (bool, error) {
		if !s.killRing.yanking {
			return true, nil
		}
		yanked := s.killRing.Yank()
		s.screen.EraseTo(s.screen.Position() - len(yanked))
		s.killRing.Rotate()
		s.screen.Insert(s.killRing.Yank()...)
		return true, nil
	},
}

// killRing implements a fixed size kill ring. When a command is described as
// killing text, the deleted text is saved for future retrieval in the kill
// ring. Consecutive kills cause the text to be accumulated in a single entry
// which can be yanked all at once. Commands which do not kill text separate the
// entries on the kill ring.
type killRing struct {
	entries []string
	killing bool
	yanking bool
}

// Append appends text to the current kill ring entry. If the previous command
// was not a kill command then a new kill ring entry is created, discarding
// the oldest entry if the max kill ring size has been reached.
func (r *killRing) Append(e string) {
	r.maybeBeginKill()
	head := len(r.entries) - 1
	r.entries[head] += e
}

// Prepend prepends text to the current kill ring entry. If the previous command
// was not a kill command then a new kill ring entry is created, discarding the
// oldest entry if the max kill ring size has been reached.
func (r *killRing) Prepend(e string) {
	r.maybeBeginKill()
	head := len(r.entries) - 1
	r.entries[head] = e + r.entries[head]
}

// Yank returns the current kill ring entry, or nil if the kill ring is empty.
func (r *killRing) Yank() []rune {
	if len(r.entries) == 0 {
		return nil
	}
	r.yanking = true
	return []rune(r.entries[len(r.entries)-1])
}

// Rotate rotates the kill ring so that the current kill ring entry becomes the
// oldest and the next newest entry becomes the current entry.
func (r *killRing) Rotate() {
	if len(r.entries) == 0 {
		return
	}
	last := r.entries[len(r.entries)-1]
	copy(r.entries[1:], r.entries)
	r.entries[0] = last
}

// Dispatch processes the specified command, clearing the killing and yanking
// states if the command is neither a kill command or a yank command.
func (r *killRing) Dispatch(s *state, cmd command, key rune) (ok bool, err error) {
	if fn, ok := killCommands[cmd]; ok {
		return fn(s, key)
	}
	r.killing = false

	if fn, ok := yankCommands[cmd]; ok {
		return fn(s, key)
	}
	r.yanking = false

	return false, nil
}

func (r *killRing) String() string {
	var buf strings.Builder
	buf.WriteString("[")
	for i := range r.entries {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(r.entries[len(r.entries)-i-1])
	}
	buf.WriteString("]")
	return buf.String()
}

func (r *killRing) maybeBeginKill() {
	if r.killing {
		return
	}
	r.killing = true

	if r.entries == nil {
		r.entries = make([]string, 0, killRingMax)
	}
	if len(r.entries) < cap(r.entries) {
		r.entries = append(r.entries, "")
	} else {
		copy(r.entries, r.entries[1:])
		r.entries[len(r.entries)-1] = ""
	}
}
