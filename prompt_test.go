package prompt

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cockroachdb/datadriven"
	"github.com/mattn/go-runewidth"
)

type mockTerm struct {
	contents []rune
	width    int
	height   int
	cursorX  int
	cursorY  int
}

var seqRE = regexp.MustCompile(`^\x1b\[(\d*)([ABCDHJKm])`)

func newMockTerm(w, h int) *mockTerm {
	return &mockTerm{
		contents: make([]rune, w*h),
		width:    w,
		height:   h,
	}
}

func (t *mockTerm) Write(p []byte) (int, error) {
	for len(p) > 0 {
		m := seqRE.FindSubmatch(p)
		if m != nil {
			var n int
			if len(m[1]) > 0 {
				var err error
				n, err = strconv.Atoi(string(m[1]))
				if err != nil {
					return -1, err
				}
			}
			// \x1b[K     erase line to right
			// \x1b[H     move cursor to 0,0
			// \x1b[2J    erase screen from cursor down
			// \x1b[<N>A  move cursor up <N>
			// \x1b[<N>B  move cursor down <N>
			// \x1b[<N>C  move cursor right <N>
			// \x1b[<N>D  move cursor left <N>
			switch m[2][0] {
			case 'A':
				t.moveUp(n)
			case 'B':
				t.moveDown(n)
			case 'C':
				t.moveRight(n)
			case 'D':
				t.moveLeft(n)
			case 'H':
				t.moveTo(0, 0)
			case 'J':
				t.eraseScreen(n)
			case 'K':
				t.eraseLine(n)
			case 'm':
				// Set attribute, ignore
			default:
				return -1, fmt.Errorf("unknown CSI command: %q", m[2][0])
			}
			p = p[len(m[0]):]
			continue
		}
		r, l := utf8.DecodeRune(p)
		if r == utf8.RuneError {
			return -1, fmt.Errorf("unable to decode utf8: [% x]", p)
		}
		t.put(r)
		p = p[l:]
	}
	return len(p), nil
}

func (t *mockTerm) String() string {
	var buf strings.Builder

	buf.WriteRune('┌')
	for x := 0; x < t.width; x++ {
		buf.WriteRune('─')
	}
	buf.WriteString("┐\n")

	for y := 0; y < t.height; y++ {
		buf.WriteRune('│')
		var prevWidth int
		for x := 0; x < t.width; x++ {
			r := t.contents[t.position(x, y)]
			if r == 0 {
				r = ' '
			}
			if prevWidth != 2 {
				buf.WriteRune(r)
			}
			if x == t.cursorX && y == t.cursorY {
				buf.WriteRune('\u0332') // combining low line
			}
			prevWidth = runewidth.RuneWidth(r)
		}
		buf.WriteString("│\n")
	}

	buf.WriteRune('└')
	for x := 0; x < t.width; x++ {
		buf.WriteRune('─')
	}
	buf.WriteRune('┘')

	return buf.String()
}

func (t *mockTerm) moveUp(n int) {
	if n == 0 {
		n = 1
	}
	t.moveTo(t.cursorX, t.cursorY-n)
}

func (t *mockTerm) moveDown(n int) {
	if n == 0 {
		n = 1
	}
	t.moveTo(t.cursorX, t.cursorY+n)
}

func (t *mockTerm) moveRight(n int) {
	if n == 0 {
		n = 1
	}
	t.moveTo(t.cursorX+n, t.cursorY)
}

func (t *mockTerm) moveLeft(n int) {
	if n == 0 {
		n = 1
	}
	t.moveTo(t.cursorX-n, t.cursorY)
}

func (t *mockTerm) moveTo(x, y int) {
	if x < 0 {
		x = 0
	} else if x > t.width {
		x = t.width
	}
	if y < 0 {
		y = 0
	} else if y > t.height {
		y = t.height
	}
	t.cursorX = x
	t.cursorY = y
}

func (t *mockTerm) eraseScreen(n int) {
	switch n {
	case 0:
		// Clear from cursor to end of screen.
		t.fill(t.cursorX, t.cursorY, t.width-t.cursorX, 1, 0)
		t.fill(0, t.cursorY+1, t.width, t.height-(t.cursorY+1), 0)
	case 1:
		// Clear from cursor to beginning of screen.
		t.fill(0, 0, t.width, t.cursorY, 0)
		t.fill(0, t.cursorY, t.cursorX, 1, 0)
	case 2:
		// Move to home, and clear from cursor to end of screen
		t.moveTo(0, 0)
		t.fill(0, 0, t.width, t.height, 0)
	}
}

func (t *mockTerm) eraseLine(n int) {
	switch n {
	case 0:
		// Clear from cursor to end of line.
		t.fill(t.cursorX, t.cursorY, t.width-t.cursorX, 1, 0)
	case 1:
		// Clear from cursor to beginning of line.
		t.fill(0, t.cursorY, t.cursorX, 1, 0)
	case 2:
		// Clear entire line.
		t.fill(0, t.cursorY, t.width, 1, 0)
	}
}

func (t *mockTerm) scroll() {
	for i := 1; i < t.height; i++ {
		copy(t.line(i-1), t.line(i))
	}
	t.fill(0, t.cursorY, t.width, 1, 0)
}

func (t *mockTerm) position(x, y int) int {
	return x + y*t.width
}

func (t *mockTerm) put(r rune) {
	switch r {
	case '\r':
		t.moveTo(0, t.cursorY)
	case '\n':
		if t.cursorY+1 < t.height {
			t.cursorY++
			return
		}
		t.cursorX = 0
		t.scroll()
	default:
		w := runewidth.RuneWidth(r)
		switch w {
		case 0:
		case 1:
			t.contents[t.position(t.cursorX, t.cursorY)] = r
			if t.cursorX+1 < t.width {
				t.cursorX++
			}
		case 2:
			if t.cursorX+2 >= t.width {
				t.cursorX = 0
				t.scroll()
			}
			pos := t.position(t.cursorX, t.cursorY)
			t.contents[pos] = r
			t.contents[pos+1] = 0
			t.cursorX += 2
		}
	}
}

func (t *mockTerm) line(y int) []rune {
	return t.contents[y*t.width : (y+1)*t.width]
}

func (t *mockTerm) fill(x, y, width, height int, r rune) {
	for i := 0; i < height; i++ {
		for j := 0; j < width; j++ {
			t.contents[t.position(x+j, y+i)] = r
		}
	}
}

func TestPrompt(t *testing.T) {
	var term *mockTerm
	var p *Prompt

	inputRE := regexp.MustCompile(`<[^>]*>`)
	inputReplacements := map[string]string{
		"<Control-a>":  string(rune(keyCtrlA)),
		"<Control-b>":  string(rune(keyCtrlB)),
		"<Control-c>":  string(rune(keyCtrlC)),
		"<Control-d>":  string(rune(keyCtrlD)),
		"<Control-e>":  string(rune(keyCtrlE)),
		"<Control-f>":  string(rune(keyCtrlF)),
		"<Control-g>":  string(rune(keyCtrlG)),
		"<Control-h>":  string(rune(keyCtrlH)),
		"<Control-k>":  string(rune(keyCtrlK)),
		"<Control-l>":  string(rune(keyCtrlL)),
		"<Control-n>":  string(rune(keyCtrlN)),
		"<Control-p>":  string(rune(keyCtrlP)),
		"<Control-r>":  string(rune(keyCtrlR)),
		"<Control-s>":  string(rune(keyCtrlS)),
		"<Control-t>":  string(rune(keyCtrlT)),
		"<Control-u>":  string(rune(keyCtrlU)),
		"<Control-w>":  string(rune(keyCtrlW)),
		"<Control-y>":  string(rune(keyCtrlY)),
		"<Meta-b>":     "\x1bb",
		"<Meta-d>":     "\x1bd",
		"<Meta-f>":     "\x1bf",
		"<Meta-t>":     "\x1bt",
		"<Meta-y>":     "\x1by",
		"<Meta-\\>":    "\x1b\\",
		"<Meta-Left>":  "\x1b\x1b[D",
		"<Meta-Right>": "\x1b\x1b[C",
		"<Meta-Enter>": "\x1b\r",
		"<Backspace>":  "\x7f",
		"<Delete>":     "\u001B[3~",
		"<Down>":       "\x1b[B",
		"<End>":        "\u001B[F",
		"<Enter>":      "\r",
		"<Home>":       "\u001B[H",
		"<Left>":       "\x1b[D",
		"<Right>":      "\x1b[C",
		"<Space>":      " ",
		"<Tab>":        "\t",
		"<Up>":         "\x1b[A",
	}
	inputReplacementFunc := func(src string) string {
		if r, ok := inputReplacements[src]; ok {
			return r
		}
		return src
	}

	inputFinished := func(text string) bool {
		text = strings.TrimSpace(text)
		return strings.HasSuffix(text, ";")
	}

	animals := []string{
		"baboon", "bat", "bear", "beaver", "bird", "bison", "boar", "bull",
		"mantis", "marmot", "mink", "mole", "monkey", "moose", "mouse", "mule",
	}

	completer := func(text []rune, wordStart, wordEnd int) []string {
		word := strings.ToLower(string(text[wordStart:wordEnd]))
		i := sort.Search(len(animals), func(i int) bool {
			return animals[i] >= word
		})
		if i >= len(animals) {
			return nil
		}
		j := i
		for ; j < len(animals); j++ {
			if !strings.HasPrefix(animals[j], word) {
				break
			}
		}
		return animals[i:j]
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		datadriven.RunTest(t, path,
			func(t *testing.T, td *datadriven.TestData) string {
				switch td.Cmd {
				case "new-term":
					if len(td.CmdArgs) != 2 {
						return fmt.Sprintf("error: new-term <width> <height>\n")
					}
					var width, height int
					td.ScanArgs(t, "width", &width)
					td.ScanArgs(t, "height", &height)
					term = newMockTerm(width, height)
					p = New(WithOutput(term), WithSize(width, height),
						WithCompleter(completer),
						WithInputFinished(inputFinished))
					p.mu.state.screen.Reset([]rune("> "))

				case "input":
					input := inputRE.ReplaceAllStringFunc(td.Input, inputReplacementFunc)
					p.inBytes = []byte(input)
					p.mu.Lock()
					defer p.mu.Unlock()
					for len(p.inBytes) > 0 {
						if result, err := p.processInputLocked(); err != nil {
							return err.Error()
						} else if len(result) > 0 {
							p.mu.state.screen.Reset([]rune("> "))
							p.mu.state.screen.Flush(p.out)
						}
					}
					return term.String()

				case "fill":
					var x, y, width, height int
					td.ScanArgs(t, "x", &x)
					td.ScanArgs(t, "y", &y)
					td.ScanArgs(t, "width", &width)
					td.ScanArgs(t, "height", &height)
					term.fill(x, y, width, height, '#')
					return term.String()
				}
				return ""
			})
	})
}
