package prompt

import "strings"

var completionCommands = map[command]commandFunc{
	cmdComplete: func(s *state, key rune) (bool, error) {
		return s.completer.Accept(s)
	},
}

// CompletionFunc is used to find the completions of the word delineated by
// [wordStart,wordEnd) within text. The full text is provided so that
// context-sensitive completion can be performed (e.g. a keyword might only be
// valid in certain contexts). The list of completions should be returned in
// priority order.
type CompletionFunc func(text []rune, wordStart, wordEnd int) []string

// completer implements tab completion. Whenever a character is inserted or
// deleted, a list of completions of the word at the current cursor position is
// computed and used to display a completion hint in dimmed text. If the hint is
// accepted the completion is inserted into the document. Any other command
// (such as cursor movement or the insertion of another character) causes the
// hint to be removed. If the new command was the insertion of another character
// a new hint will be displayed.
type completer struct {
	// fn is the completion function to invoke to compute completions of the word at
	// the cursor position.
	fn CompletionFunc
	// wordStart and wordEnd are the start and end position of the word being
	// completed.
	wordStart int
	wordEnd   int
	// prefix is the prefix of the completions that matched the word at the cursor
	// position.
	prefix []rune
	// suffix the displayed completion text.
	suffix []rune
	// shared is the portion of suffix that is identical across the completions and
	// which will be accepted when a complete command is invoked.
	shared int
}

// Try performs completion of the word at the current cursor position.
func (c *completer) Try(s *state) {
	if c.fn == nil {
		// No completion callback specified.
		return
	}

	// Determine if there is a word underneath the current cursor position.
	text := s.screen.Text()
	pos := s.screen.Position()
	wordStart := s.screen.PrevWordStart(pos)
	wordEnd := s.screen.NextWordEnd(wordStart)
	if wordStart == wordEnd || pos < wordStart || pos > wordEnd {
		return
	}

	completions := c.fn(text, wordStart, wordEnd)
	if len(completions) == 0 {
		return
	}

	// Compute the shared prefix of the completions.
	shared := len(completions[0])
	for i := 1; i < len(completions); i++ {
		n := shared
		if n > len(completions[i]) {
			n = len(completions[i])
		}
		if n > len(completions[i-1]) {
			n = len(completions[i-1])
		}
		shared = 0
		for shared < n && completions[i][shared] == completions[i-1][shared] {
			shared++
		}
	}

	// Compute the completion hint to display. If there are multiple completions we
	// display the alternatives as a comma separated listed up to a length of 20
	// characters, after which we display "...". We always display at least 1
	// completion in full.
	var suffix strings.Builder
	for i := range completions {
		if i > 0 {
			if len(completions[i])+suffix.Len() > 20 {
				suffix.WriteString("...")
				break
			}
			suffix.WriteString(",")
			suffix.WriteString(completions[i])
		} else {
			suffix.WriteString(completions[i][wordEnd-wordStart:])
		}
	}

	// The prefix of the completion. This should be equivalent to the word being
	// completed modulo capitalization.
	c.prefix = []rune(completions[0][:wordEnd-wordStart])
	c.suffix = []rune(suffix.String())
	c.wordStart = wordStart
	c.wordEnd = wordEnd
	c.shared = shared - (wordEnd - wordStart)

	s.screen.MoveTo(c.wordEnd)
	// TODO(peter): attrDim doesn't seem to be supported on Warp. Perhaps it isn't
	// supported on other terminals.
	s.screen.SetAttrs(attrDim)
	s.screen.Insert(c.suffix...)
	s.screen.SetAttrs("")
	s.screen.MoveTo(pos)
}

// Accept accepts the currently displayed completion hint. After accepting the
// completion, or if there was no completion hint currently displayed, another
// attempt is made to perform completion at the cursor position.
func (c *completer) Accept(s *state) (ok bool, err error) {
	if c.suffix != nil {
		s.screen.MoveTo(c.wordStart)
		s.screen.EraseTo(c.wordEnd + len(c.suffix))
		s.screen.Insert(c.prefix...)
		s.screen.Insert(c.suffix[:c.shared]...)
		c.prefix = nil
		c.suffix = nil
	}
	c.Try(s)
	return true, nil
}

// Cancel cancels the current completion, restoring the display to the
// pre-completion hint state.
func (c *completer) Cancel(s *state) {
	if c.suffix == nil {
		return
	}
	pos := s.screen.Position()
	s.screen.MoveTo(c.wordEnd)
	s.screen.EraseTo(c.wordEnd + len(c.suffix))
	s.screen.MoveTo(pos)
	c.prefix = nil
	c.suffix = nil
}

// Dispatch processes the specified command, canceling the current completion if
// the command is not a completion command.
func (c *completer) Dispatch(s *state, cmd command, key rune) (ok bool, err error) {
	if fn, ok := completionCommands[cmd]; ok {
		return fn(s, key)
	}
	c.Cancel(s)
	return false, nil
}
