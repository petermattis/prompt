package prompt

import (
	"bytes"
	"io"
	"strconv"
	"unicode"

	"github.com/mattn/go-runewidth"
)

// TODO(peter): scroll input that is taller than the screen
// TODO(peter): continuation prompt?
// TODO(peter): syntax highlighting?

const (
	attrBold      = "\x1b[1m"
	attrDim       = "\x1b[2m"
	attrReset     = "\x1b[0m"
	attrReverse   = "\x1b[7m"
	attrUnderline = "\x1b[4m"
)

const (
	fgDefault   = "\x1b[39m"
	fgBlack     = "\x1b[30m"
	fgBlue      = "\x1b[94m"
	fgBrown     = "\x1b[33m"
	fgCyan      = "\x1b[36m"
	fgDarkBlue  = "\x1b[34m"
	fgDarkGray  = "\x1b[90m"
	fgDarkGreen = "\x1b[32m"
	fgDarkRed   = "\x1b[31m"
	fgFuchsia   = "\x1b[95m"
	fgGreen     = "\x1b[92m"
	fgLightGray = "\x1b[37m"
	fgPurple    = "\x1b[35m"
	fgRed       = "\x1b[91m"
	fgTurquoise = "\x1b[96m"
	fgWhite     = "\x1b[97m"
	fgYellow    = "\x1b[93m"
)

const (
	bgDefault   = "\x1b[49m"
	bgBlack     = "\x1b[40m"
	bgBlue      = "\x1b[104m"
	bgBrown     = "\x1b[43m"
	bgCyan      = "\x1b[46m"
	bgDarkBlue  = "\x1b[44m"
	bgDarkGray  = "\x1b[100m"
	bgDarkGreen = "\x1b[42m"
	bgDarkRed   = "\x1b[41m"
	bgFuchsia   = "\x1b[105m"
	bgGreen     = "\x1b[102m"
	bgLightGray = "\x1b[47m"
	bgPurple    = "\x1b[45m"
	bgRed       = "\x1b[101m"
	bgTurquoise = "\x1b[106m"
	bgWhite     = "\x1b[107m"
	bgYellow    = "\x1b[103m"
)

// lineInfo holds the state for a single line of displayed text.
type lineInfo struct {
	// startPos and endPos specify the range of text displayed on the line as
	// screen.text[startPos:endPos].
	startPos int
	endPos   int
	// x and y specify the coordinates of the line.
	x, y int
}

// screen models a prompt, input text, and the display of the prompt and text on
// a terminal. Rendering assumes support for a minimal set of ANSI escape
// sequences: relative cursor movement (ESC[<num>{A,B,C,D}), move to top left
// corner (ESC[H), erase screen (ESC[2J), and erase line to right (ESC[K),
type screen struct {
	// prefix holds text to display before the input text.
	prefix []rune
	// suffix holds text to display after the input text.
	suffix []rune
	// text holds the text to be displayed. The prompt is stored as a prefix of the
	// text, and suffix is stored as a suffix. The user input is stored in
	// text[len(prefix):len(text)-len(suffix)] which can be retrieved using the
	// Text() method.
	text []rune
	// lines holds cached information about the rendered lines. Each line is a
	// single row in the terminal. If the input text is too wide to fit on a single
	// line, it is wrapped. The input text is split into multiple lines on newlines
	// ('\n').
	lines []lineInfo
	// width is the width in characters of the terminal.
	width int
	// height is the height in characters of the terminal.
	height int
	// cursorPos is the index within text denoting the cursor's position. Always in the
	// range [len(prefix), len(text)-len(suffix)].
	cursorPos int
	// cursorX is the 0-indexed horizontal position of the cursor from the left side
	// of the terminal.
	cursorX int
	// cursorY is the 0-indexed vertical position of the cursor from the top of the
	// terminal.
	cursorY int
	// maxY is the maximum row that has been rendered.
	maxY int
	// outbuf holds the buffered text to send to the terminal.
	outbuf bytes.Buffer
}

func (s *screen) Init() {
	// These defaults are usually override by SetSize().
	s.width = 80
	s.height = 40
}

// Flush writes the buffered drawing commands to the specified writer and clears
// the buffer.
func (s *screen) Flush(w io.Writer) {
	debugPrintf("output: %q\n", s.outbuf.Bytes())
	_, _ = io.Copy(w, &s.outbuf)
	s.outbuf.Reset()
}

// Reset resets the buffer to read new input.
func (s *screen) Reset(prefix []rune) {
	s.prefix = prefix
	s.suffix = nil
	s.text = append([]rune(nil), s.prefix...)
	s.lines = nil
	s.cursorPos = 0
	s.cursorX = 0
	s.cursorY = 0
	s.maxY = 0
	s.renderText(len(s.text))
	s.MoveTo(0)
}

// Cancel cancels the current input, leaving it on screen, and resets state to
// read a new input.
func (s *screen) Cancel() {
	s.MoveTo(len(s.text))
	if s.cursorX != 0 {
		s.outbuf.WriteString("\r\n")
	}
	s.Reset(s.prefix)
}

// SetSize sets the width and height of the screen and re-renders the display to
// account for the new size.
func (s *screen) SetSize(width, height int) {
	if s.width == 0 {
		s.width, s.height = width, height
		return
	}

	if width == 0 {
		width = 1
	}

	oldWidth := s.width
	s.width, s.height = width, height

	switch {
	case width == oldWidth:
		return

	case width < oldWidth:
		// Some terminals truncate lines that were too long when horizontally shrinking
		// the terminal. Others will attempt to wrap them. In the wrapping case, we're
		// unsure of how many new lines were added and where the cursor ended up, so we
		// just refresh the entire screen.
		s.Refresh()
		return

	case width > oldWidth:
		lines := s.maxY
		s.cursorX = width
		s.invalidateLines()
		savedPos := s.cursorPos - len(s.prefix)
		s.cursorPos = 0
		s.moveCursor(0, 0)
		s.renderText(len(s.text))
		s.eraseLineToRight()
		for s.cursorY < lines {
			s.moveCursor(0, s.cursorY+1)
			s.eraseLineToRight()
		}
		s.MoveTo(savedPos)
	}
}

// SetSuffix sets the suffix to display. The suffix is displayed after the input
// text and is used to display the search history prompt.
func (s *screen) SetSuffix(newSuffix []rune) {
	oldSuffix := s.suffix
	s.suffix = newSuffix

	s.text = s.text[:len(s.text)-len(oldSuffix)]
	if len(s.text)+len(newSuffix) > cap(s.text) {
		newText := make([]rune, len(s.text), 2*(len(s.text)+len(newSuffix)))
		copy(newText, s.text)
		s.text = newText
	}
	pos := len(s.text)
	s.text = s.text[:len(s.text)+len(newSuffix)]
	copy(s.text[pos:], newSuffix)

	savedPos := s.cursorPos - len(s.prefix)
	s.invalidateLines()
	s.MoveTo(len(s.text))
	s.renderText(len(s.text))
	s.eraseLineToRight()
	for ; s.cursorY < s.maxY; s.cursorY++ {
		s.outbuf.WriteString("\r\n")
		s.cursorX = 0
		s.eraseLineToRight()
	}
	s.MoveTo(savedPos)
}

// Refresh clears the screen and redraws the prompt and text.
func (s *screen) Refresh() {
	s.eraseScreen()
	s.invalidateLines()
	savedPos := s.cursorPos - len(s.prefix)
	s.cursorPos = 0
	s.cursorX, s.cursorY = 0, 0
	s.renderText(len(s.text))
	s.MoveTo(savedPos)
}

// MoveTo moves the cursor to the specified position.
func (s *screen) MoveTo(pos int) {
	s.maybeRecomputeLines()

	if pos < 0 {
		pos = 0
	}
	if pos > len(s.text)-len(s.suffix)-len(s.prefix) {
		pos = len(s.text) - len(s.suffix) - len(s.prefix)
	}
	pos += len(s.prefix)

	var l *lineInfo
	for i := 0; i < len(s.lines); i++ {
		if pos <= s.lines[i].endPos {
			l = &s.lines[i]
			break
		}
	}

	_, width, _ := fitGraphemes(s.text[l.startPos:pos], s.width-l.x)
	x := l.x + width
	y := l.y + x/s.width
	x = x % s.width

	s.cursorPos = pos
	s.moveCursor(x, y)
}

// Insert inserts text at the current cursor position, moving the cursor
// forwards.
func (s *screen) Insert(text ...rune) {
	origText := text
	text = text[:0]
	for _, r := range origText {
		if isPrintable(r) {
			text = append(text, r)
		}
	}

	if len(text) < len(origText) {
		s.outbuf.WriteRune(keyCtrlG) // ctrl-G == bell/beep
	}
	if len(text) == 0 {
		return
	}

	s.invalidateLines()
	if len(s.text)+len(text) > cap(s.text) {
		newText := make([]rune, len(s.text), 2*(len(s.text)+len(text)))
		copy(newText, s.text)
		s.text = newText
	}
	s.text = s.text[:len(s.text)+len(text)]
	copy(s.text[s.cursorPos+len(text):], s.text[s.cursorPos:])
	copy(s.text[s.cursorPos:], text)
	newPos := s.cursorPos + len(text) - len(s.prefix)
	s.renderText(len(s.text))
	s.MoveTo(newPos)
}

// EraseTo erase the characters from the current cursor position to the target
// position, adjusting the cursor position to account for the deleted text.
func (s *screen) EraseTo(pos int) string {
	if pos < 0 {
		pos = 0
	}
	if pos > len(s.text)-len(s.suffix)-len(s.prefix) {
		pos = len(s.text) - len(s.suffix) - len(s.prefix)
	}
	pos += len(s.prefix)

	var erased string
	switch {
	case pos == s.cursorPos:
		return ""
	case pos < s.cursorPos:
		erased = string(s.text[pos:s.cursorPos])
		copy(s.text[pos:], s.text[s.cursorPos:])
		s.text = s.text[:len(s.text)-(s.cursorPos-pos)]
		s.MoveTo(pos - len(s.prefix))
	case pos > s.cursorPos:
		erased = string(s.text[s.cursorPos:pos])
		copy(s.text[s.cursorPos:], s.text[pos:])
		s.text = s.text[:len(s.text)-(pos-s.cursorPos)]
	}

	s.invalidateLines()
	newPos := s.cursorPos - len(s.prefix)
	s.renderText(len(s.text))

	s.eraseLineToRight()
	for ; s.cursorY < s.maxY; s.cursorY++ {
		s.outbuf.WriteString("\r\n")
		s.cursorX = 0
		s.eraseLineToRight()
	}
	s.MoveTo(newPos)
	return erased
}

// End returns the position of the end of the input text.
func (s *screen) End() int {
	return len(s.text)
}

// Text returns the current input text. Note that the returned value points to
// the underlying storage used by the screen and should not be modified.
func (s *screen) Text() []rune {
	return s.text[len(s.prefix) : len(s.text)-len(s.suffix)]
}

// Position returns the current cursor position within the input.
func (s *screen) Position() int {
	return s.cursorPos - len(s.prefix)
}

// NextGraphemeEnd returns the position of the end of the next grapheme after
// the current cursor position, accounting for zero-width characters.
func (s *screen) NextGraphemeEnd() int {
	text := s.Text()
	pos := s.cursorPos - len(s.prefix)
	if pos >= len(s.text) {
		return pos
	}

	for n := 0; n < 1 && pos < len(text); pos++ {
		if text[pos] == '\n' || runewidth.RuneWidth(text[pos]) != 0 {
			n++
		}
	}
	for pos < len(text) && text[pos] != '\n' && runewidth.RuneWidth(text[pos]) == 0 {
		pos++
	}
	return pos
}

// PrevGraphemeStart returns the position of the start of the previous grapheme
// before the current cursor position, accounting for zero-width characters.
func (s *screen) PrevGraphemeStart() int {
	if s.cursorPos <= len(s.prefix) {
		return 0
	}

	text := s.Text()[:s.cursorPos-len(s.prefix)]
	pos := len(text)
	for n := 0; n < 1 && pos > 0; pos-- {
		if text[pos-1] == '\n' || runewidth.RuneWidth(text[pos-1]) != 0 {
			n++
		}
	}
	return pos
}

// NextWordEnd returns the position of the end of the next word after the
// current cursor position.
func (s *screen) NextWordEnd(pos int) int {
	text := s.Text()
	// Advance to the start of the next word.
	for pos < len(text) {
		if isWord(text[pos]) {
			break
		}
		pos++
	}
	// Advance to the end of the next word.
	for pos < len(text) {
		if !isWord(text[pos]) {
			break
		}
		pos++
	}
	return pos
}

// PrevWordStart returns the position of the start of the previous word before
// the current cursor position.
func (s *screen) PrevWordStart(pos int) int {
	text := s.Text()
	pos--
	// Advance to the end of the previous word.
	for pos > 0 {
		if isWord(text[pos]) {
			break
		}
		pos--
	}
	// Advance to the start of the previous word.
	for pos > 0 {
		if !isWord(text[pos-1]) {
			break
		}
		pos--
	}
	if pos < 0 {
		return 0
	}
	return pos
}

func (s *screen) maybeRecomputeLines() {
	if s.lines != nil {
		return
	}

	var pos int
	var x, y int
	s.lines = nil

	for text := s.text; len(text) >= 0; {
		s.lines = append(s.lines, lineInfo{
			startPos: pos,
			endPos:   pos,
			x:        x,
			y:        y,
		})
		if len(text) == 0 {
			break
		}

		consumed, width, newline := fitGraphemes(text, s.width-x)
		x += width
		y += x / s.width
		x = x % s.width

		l := &s.lines[len(s.lines)-1]
		l.endPos = pos + consumed

		text = text[consumed:]
		pos += consumed

		if newline || consumed == 0 {
			x = 0
			y++
			if newline {
				pos++
				text = text[1:]
			}
		}
	}

	if s.maxY < y {
		s.maxY = y
	}
}

func (s *screen) invalidateLines() {
	s.lines = nil
}

// renderText renders the range of text [b.cursorPos,end) advancing the cursor
// position to end.
func (s *screen) renderText(end int) {
	for text := s.text[s.cursorPos:end]; len(text) > 0; {
		consumed, width, newline := fitGraphemes(text, s.width-s.cursorX)
		for _, r := range text[:consumed] {
			s.outbuf.WriteRune(r)
		}
		s.cursorPos += consumed
		text = text[consumed:]

		if width > 0 {
			s.cursorX += width
			s.cursorY += s.cursorX / s.width
			s.cursorX = s.cursorX % s.width
			if s.cursorX == 0 {
				// Normally terminals will advance the current position when writing a
				// character. But that doesn't happen for the last character in a line. However,
				// when writing a character (except a new line) that causes a line wrap, the
				// position will be advanced two places.
				//
				// So, if we are stopping at the end of a line, we need to write a newline so
				// that our cursor can be advanced to the next line.
				s.outbuf.WriteString("\r\n")
			}
		}

		if newline || consumed == 0 {
			s.eraseLineToRight()
			s.outbuf.WriteString("\r\n")
			s.cursorX = 0
			s.cursorY++
			if newline {
				s.cursorPos++
				text = text[1:]
			}
		}
	}
}

func (s *screen) moveCursor(x, y int) {
	const (
		csi             = "\x1b[" // csi = Control Sequence Introducer
		moveUpSuffix    = "A"
		moveDownSuffix  = "B"
		moveRightSuffix = "C"
		moveLeftSuffix  = "D"
	)

	if y < s.cursorY {
		up := s.cursorY - y
		if up == 1 {
			s.outbuf.WriteString(csi)
			s.outbuf.WriteString(moveUpSuffix)
		} else if up > 1 {
			s.outbuf.WriteString(csi)
			s.outbuf.WriteString(strconv.Itoa(up))
			s.outbuf.WriteString(moveUpSuffix)
		}
	}

	if y > s.cursorY {
		down := y - s.cursorY
		if down == 1 {
			s.outbuf.WriteString(csi)
			s.outbuf.WriteString(moveDownSuffix)
		} else if down > 1 {
			s.outbuf.WriteString(csi)
			s.outbuf.WriteString(strconv.Itoa(down))
			s.outbuf.WriteString(moveDownSuffix)
		}
	}

	if x < s.cursorX {
		left := s.cursorX - x
		if left == 1 {
			s.outbuf.WriteString(csi)
			s.outbuf.WriteString(moveLeftSuffix)
		} else if left > 1 {
			s.outbuf.WriteString(csi)
			s.outbuf.WriteString(strconv.Itoa(left))
			s.outbuf.WriteString(moveLeftSuffix)
		}
	}

	if x > s.cursorX {
		right := x - s.cursorX
		if right == 1 {
			s.outbuf.WriteString(csi)
			s.outbuf.WriteString(moveRightSuffix)
		} else if right > 1 {
			s.outbuf.WriteString(csi)
			s.outbuf.WriteString(strconv.Itoa(right))
			s.outbuf.WriteString(moveRightSuffix)
		}
	}

	s.cursorX = x
	s.cursorY = y
}

// eraseLineToRight generates the escape sequence to erase the line from the
// current cursor position to the end of the line.
func (s *screen) eraseLineToRight() {
	s.outbuf.WriteString("\x1b[K")
}

// eraseScreen generates the escape sequence to move the cursor to the top left
// of the screen and to erase the contents of the screen.
func (s *screen) eraseScreen() {
	s.outbuf.WriteString("\x1b[H\x1b[2J")
}

const zeroWidthJoiner = '\u200d'

func isPrintable(key rune) bool {
	if (key & (keyCtrl | keyAlt)) != 0 {
		return false
	}
	// TODO(peter): should zeroWidthJoiner sequences be handled? Doing so is
	// necessary to handle multi-character emojis such as "woman kissing woman"
	// ("👩‍❤️‍💋‍👩"), which is 8 code points. Most readline libraries seem to get
	// this wrong (including GNU readline), but terminal support for such emojis is
	// weak.
	if key == zeroWidthJoiner {
		return false
	}
	isInSurrogateArea := key >= 0xd800 && key <= 0xdbff
	return key == '\n' || key >= 32 && !isInSurrogateArea
}

func isWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func fitGraphemes(s []rune, avail int) (consumed, width int, newline bool) {
	for i, r := range s {
		if r == '\n' {
			return i, width, true
		}
		if r < 127 {
			if width >= avail {
				return i, width, false
			}
			width++
			continue
		}
		switch runewidth.RuneWidth(r) {
		case 0:
		case 1:
			if width >= avail {
				return i, width, false
			}
			width++
		case 2:
			if width+2 >= avail {
				return i, width, false
			}
			width += 2
		}
	}
	return len(s), width, false
}
